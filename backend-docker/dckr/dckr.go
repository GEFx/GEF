package dckr

import (
	"bytes"
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	//"io/ioutil"
	"log"
	"os"
	//"path/filepath"
	"strconv"
	"strings"
	"io"
	//"archive/tar"
	"archive/tar"
	"io/ioutil"
)

const (
	minimalDockerVersion = 1006 // major * 1000 + minor
)

// Config configuration for building docker clients
type Config struct {
	UseBoot2Docker bool
	Endpoint       string
	Description    string
}

type tarInputStream struct {
	*bytes.Buffer
}

func (c Config) String() string {
	if c.Endpoint != "" {
		return fmt.Sprintf("Endpoint: %s -- %s", c.Endpoint, c.Description)
	} else if c.UseBoot2Docker {
		return fmt.Sprintf("Boot2Docker[env] -- %s", c.Description)
	}
	return fmt.Sprintf("unknown -- %s", c.Description)
}

// Client a Docker client with easy to use API
type Client struct {
	cfg Config
	c   *docker.Client
}

// ImageID is a type for docker image ids
type ImageID string

// ContainerID is a type for docker image ids
type ContainerID string

// VolumeID is a type for docker volume ids
type VolumeID string

//VolumeMountpoint is a type for docker volume mountpoint
type VolumeMountpoint string

// Image is a struct for Docker images
type Image struct {
	ID      ImageID
	RepoTag string
	Labels  map[string]string
}

// Container is a struct for Docker containers
type Container struct {
	ID     ContainerID
	Image  Image
	State  docker.State
	Mounts []docker.Mount
}

type Volume struct {
	ID         VolumeID
	Mountpoint VolumeMountpoint
}

// NewClientFirstOf returns a new docker client or an error
func NewClientFirstOf(cfg []Config) (Client, error) {
	var buf bytes.Buffer
	for _, dcfg := range cfg {
		client, err := NewClient(dcfg)
		if err != nil || client.c == nil {
			buf.WriteString(fmt.Sprintf(
				"%s:\n\t%s\nReason:%s\n",
				"Failed to make new docker client using configuration",
				dcfg, err))
		} else if client.c != nil {
			version, err := checkForMinimalDockerVersion(client.c)
			if err != nil {
				buf.WriteString(fmt.Sprintf(
					"%s:\n\t%s\nReason:%s\n",
					"Docker server version check has failed",
					dcfg, err))
			} else {
				log.Println("Successfully created Docker client using config:", dcfg)
				log.Println("Docker server version:", version)
				return client, nil
			}
		}
	}
	return Client{}, errors.New(buf.String())
}

// NewClient returns a new docker client or an error
func NewClient(dcfg Config) (Client, error) {
	var client *docker.Client
	var err error
	if dcfg.Endpoint != "" {
		client, err = docker.NewClient(dcfg.Endpoint)
	} else if dcfg.UseBoot2Docker {
		endpoint := os.Getenv("DOCKER_HOST")
		if endpoint != "" {
			path := os.Getenv("DOCKER_CERT_PATH")
			cert := fmt.Sprintf("%s/cert.pem", path)
			key := fmt.Sprintf("%s/key.pem", path)
			ca := fmt.Sprintf("%s/ca.pem", path)
			client, err = docker.NewTLSClient(endpoint, cert, key, ca)
		}
	} else {
		return Client{}, errors.New("empty docker configuration")
	}
	if err != nil || client == nil {
		return Client{dcfg, client}, err
	}

	return Client{dcfg, client}, client.Ping()
}

func checkForMinimalDockerVersion(c *docker.Client) (string, error) {
	env, err := c.Version()
	if err != nil {
		return "", err
	}
	m := env.Map()
	version := m["Version"]
	arr := strings.Split(version, ".")
	if len(arr) < 2 {
		return "", fmt.Errorf("unparsable version string: %s", version)
	}
	major, err := strconv.Atoi(arr[0])
	if err != nil {
		return "", fmt.Errorf("unparsable major version: %s", version)
	}
	minor, err := strconv.Atoi(arr[1])
	if err != nil {
		return "", fmt.Errorf("unparsable minor version: %s", version)
	}
	if major*1000+minor < minimalDockerVersion {
		return "", fmt.Errorf("unusably old Docker version: %s", version)
	}
	return version, nil
}

// IsValid returns true if the client is connected
func (c Client) IsValid() bool {
	return c.c != nil && c.c.Ping() == nil
}

func makeImage(img *docker.Image) Image {
	repoTag := ""
	if len(img.RepoTags) > 0 {
		repoTag = img.RepoTags[0]
	}
	var labels map[string]string
	if img.Config != nil {
		labels = img.Config.Labels
	}
	return Image{
		ID:      ImageID(img.ID),
		RepoTag: repoTag,
		Labels:  labels,
	}
}

func makeImage2(img docker.APIImages) Image {
	repoTag := ""
	if len(img.RepoTags) > 0 {
		repoTag = img.RepoTags[0]
	}
	return Image{
		ID:      ImageID(img.ID),
		RepoTag: repoTag,
		Labels:  img.Labels,
	}
}

// InspectImage returns the image stats
func (c Client) InspectImage(id ImageID) (Image, error) {
	img, err := c.c.InspectImage(string(id))
	if err != nil {
		return Image{}, err
	}
	return makeImage(img), err
}

// ListImages lists the docker images
func (c Client) ListImages() ([]Image, error) {
	imgs, err := c.c.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		return nil, err
	}
	ret := make([]Image, 0, 0)
	for _, img := range imgs {
		ret = append(ret, makeImage2(img))
	}
	return ret, nil
}

// BuildImage builds a Docker image from a directory with a Dockerfile
func (c *Client) BuildImage(dirpath string) (Image, error) {
	var buf bytes.Buffer
	err := c.c.BuildImage(docker.BuildImageOptions{
		Dockerfile:   "Dockerfile",
		ContextDir:   dirpath,
		OutputStream: &buf,
	})
	var img Image
	if err != nil {
		return img, err
	}
	stepPrefix := "Step "
	successPrefix := "Successfully built "
	for err == nil {
		var line string
		line, err = buf.ReadString('\n')
		// fmt.Printf("build line: `%s`", line)
		if strings.HasPrefix(line, stepPrefix) {
			// step++
		} else if strings.HasPrefix(line, successPrefix) {
			img.ID = ImageID(strings.TrimSpace(line[len(successPrefix):]))
		}
	}
	err = c.c.Ping()
	if err != nil {
		log.Println("Warning: docker client lost after building image: ", err)
		var nc Client
		nc, err = NewClient(c.cfg)
		if err != nil {
			return img, err
		}
		c.c = nc.c
	}
	if err != nil && img.ID == "" {
		err = errors.New("unknown docker failure")
	}
	return c.InspectImage(img.ID)
}

// ExecuteImage takes a docker image, creates a container and executes it
func (c Client) ExecuteImage(id ImageID, binds []string) (ContainerID, error) {
	img, err := c.c.InspectImage(string(id))
	if err != nil {
		return ContainerID(""), err
	}

	hc := docker.HostConfig{
		Binds: binds,
	}
	cco := docker.CreateContainerOptions{
		Name:       "",
		Config:     img.Config,
		HostConfig: &hc,
	}

	cont, err := c.c.CreateContainer(cco)
	if err != nil {
		return ContainerID(""), err
	}

	err = c.c.StartContainer(cont.ID, &hc)
	if err != nil {
		c.c.RemoveContainer(docker.RemoveContainerOptions{ID: cont.ID, Force: true})
		return ContainerID(""), err
	}

	return ContainerID(cont.ID), nil
}

// DeleteImage removes an image by ID
func (c Client) DeleteImage(id string) (error) {
	err := c.c.RemoveImage(id)
	return err
}

// StartExitedContainer starts an existing container
func (c Client) StartExistingContainer(contID string, binds []string) (ContainerID, error) {
	hc := docker.HostConfig{
		Binds: binds,
	}

	err := c.c.StartContainer(contID, &hc)
	if err != nil {
		c.c.RemoveContainer(docker.RemoveContainerOptions{ID: contID, Force: true})
		return ContainerID(""), err
	}
	return ContainerID(contID), nil
}

// WaitContainer takes a docker container and waits for its finish.
// It returns the exit code of the container.
func (c Client) WaitContainer(id ContainerID, removeOnExit bool) (int, error) {
	containerID := string(id)
	exitCode, err := c.c.WaitContainer(containerID)
	if removeOnExit {
		c.c.RemoveContainer(docker.RemoveContainerOptions{ID: containerID, Force: true})
	}
	return exitCode, err
}

// ListContainers lists the docker images
func (c Client) ListContainers() ([]Container, error) {
	conts, err := c.c.ListContainers(
		docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}
	ret := make([]Container, 0, 0)
	for _, cont := range conts {
		img, _ := c.InspectImage(ImageID(cont.Image))
		mounts := make([]docker.Mount, 0, 0)
		for _, cont := range cont.Mounts {
			mounts = append(mounts, docker.Mount{
				Name:        cont.Name,
				Source:      cont.Source,
				Destination: cont.Destination,
				Driver:      cont.Driver,
				RW:          cont.RW,
				Mode:        cont.Mode,
			})
		}
		ret = append(ret, Container{
			ID:    ContainerID(cont.ID),
			Image: img,
			State: docker.State{
				Status: cont.Status,
			},
			Mounts: mounts,
		})
	}
	return ret, nil
}

// InspectContainer returns the container details
func (c Client) InspectContainer(id ContainerID) (Container, error) {
	cont, err := c.c.InspectContainer(string(id))
	img, _ := c.InspectImage(ImageID(cont.Image))
	ret := Container{
		ID:     ContainerID(cont.ID),
		Image:  img,
		State:  cont.State,
		Mounts: cont.Mounts,
	}
	if err != nil {
		return ret, err
	}
	return ret, err
}

// ListVolumes list all named volumes
func (c Client) ListVolumes() ([]Volume, error) {
	vols, err := c.c.ListVolumes(docker.ListVolumesOptions{})
	if err != nil {
		return nil, err
	}

	ret := make([]Volume, 0, 0)

	for _, vol := range vols {
		volume, _ := c.InspectVolume(VolumeID(vol.Name))
		ret = append(ret, Volume{
			ID:         VolumeID(volume.ID),
			Mountpoint: VolumeMountpoint(volume.Mountpoint),
		})
	}
	return ret, nil

}

// InspectVolume returns the volume details
func (c Client) InspectVolume(id VolumeID) (Volume, error) {
	volume, err := c.c.InspectVolume(string(id))
	ret := Volume{
		ID:         VolumeID(volume.Name),
		Mountpoint: VolumeMountpoint(volume.Mountpoint),
	}
	return ret, err
}

// return file names in a directory
// TODO
func (c Client) ListVolumeContent(id VolumeID) []string {
	return nil
}

// BuildVolume creates a volume, copies data from dirpath
func (c Client) BuildVolume(dirpath string) (Volume, error) {
	volume, err := c.c.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return Volume{}, err
	}
	ret := Volume{
		ID:         VolumeID(volume.Name),
		Mountpoint: VolumeMountpoint(volume.Mountpoint),
	}
	//copy all content to volume
	//log.Println(dirpath, string(ret.Mountpoint))
	//err = copyDataToVolume(dirpath, string(ret.Mountpoint))
	return ret, err
}

//RemoveVolume removes a volume
func (c Client) RemoveVolume(id VolumeID) error {
	err := c.c.RemoveVolume(string(id))
	return err
}


func (c Client) GetTarStream(containerID, filePath string) (io.ReadCloser, error) {
	preader, pwriter := io.Pipe()
	opts := docker.DownloadFromContainerOptions{
		Path:         filePath,
		OutputStream: pwriter,
	}

	go func() {
		defer pwriter.Close()
		fmt.Println("Requesting file", opts.Path)
		if err := c.c.DownloadFromContainer(containerID, opts); err != nil {
			log.Println(filePath + " has been retrieved")
			//return preader, err
		}
	}()


	/*defer pwriter.Close()
	log.Println("Requesting file", opts.Path)
	if err := c.c.DownloadFromContainer(containerID, opts); err != nil {
		log.Println(filePath + " has been retrieved")
		return preader, err
	}*/
	log.Println(filePath + " has not been retrieved")

	return preader, nil
}


func (c Client) UploadSingleFile(containerID, filePath string) (error) {
	var b bytes.Buffer
	fileHandler, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Cannot open " + filePath +  ": " + err.Error())
		return err
	}

	tw := tar.NewWriter(&b)
	header, err := tar.FileInfoHeader(fileHandler, "")

	err = tw.WriteHeader(header)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	_, err = tw.Write(contents)
	if err != nil {
		log.Println("write to a file" + err.Error())
		return err
	}

	opts := docker.UploadToContainerOptions{
		Path: "/root/",
		InputStream: &b,
	}

	err = c.c.UploadToContainer(containerID, opts)
	return err
}
