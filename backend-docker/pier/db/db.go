package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/gorp.v1"

	"bytes"
	"fmt"
	"github.com/EUDAT-GEF/GEF/backend-docker/pier/internal/dckr"
	"github.com/pborman/uuid"
	"log"

	"strconv"
	"strings"
	"time"
)

type VolumeID dckr.VolumeID

type Db struct{ gorp.DbMap }

// Job stores the information about a service execution
type Job struct {
	ID           JobID
	ServiceID    ServiceID
	Input        string
	Created      time.Time
	State        *JobState
	InputVolume  VolumeID
	OutputVolume VolumeID
	Tasks        []Task
}

type JobTable struct {
	ID           string
	ServiceID    string
	Input        string
	Created      time.Time
	Error  string
	Status string
	Code   int
	InputVolume  string
	OutputVolume string
	Revision     int
}

// JobState exported
type JobState struct {
	Error  string
	Status string
	Code   int
}

// JobID exported
type JobID string

type jobArray []Job

// Task contains tasks related to a specific job
type Task struct {
	ID            string
	Name          string
	ContainerID   dckr.ContainerID
	Error         string
	ExitCode      int
	ConsoleOutput string
}

// TaskTable is used for the SQL database
type TaskTable struct {
	ID            string
	Name          string
	ContainerID   string
	Error         string
	ExitCode      int
	ConsoleOutput string
	Revision      int
	JobID         string
}

// LatestOutput used to serialize console output to json
type LatestOutput struct {
	Name          string
	ConsoleOutput string
}

// Bind describes the binding between an IOPort and a docker volume
type Bind struct {
	IOPort   IOPort
	VolumeID dckr.VolumeID
}

// GefSrvLabelPrefix is the prefix identifying GEF related labels
const GefSrvLabelPrefix = "eudat.gef.service."
const GorpVersionColumn = "Revision"

// Service describes metadata for a GEF service
type Service struct {
	ID          ServiceID
	ImageID     dckr.ImageID
	Name        string
	RepoTag     string
	Description string
	Version     string
	Created     time.Time
	Size        int64
	Input       []IOPort
	Output      []IOPort
}

// ServiceTable is used for the SQL database
type ServiceTable struct {
	ID          string
	ImageID     string
	Name        string
	RepoTag     string
	Description string
	Version     string
	Revision    int
	Created     time.Time
	Size        int64
}

// ServiceID exported
type ServiceID string

// IOPort is an i/o specification for a service
// The service can only read data from volumes and write to a single volume
// Path specifies where the volumes are mounted
type IOPort struct {
	ID   string
	Name string
	Path string
}

// IOPortTable is used for the SQL database
type IOPortTable struct {
	ID        string
	Name      string
	Path      string
	IsInput   bool
	Revision  int
	ServiceID string
}

// InitDb exported
func InitDb() (Db, error) {
	var dataBaseHandler Db
	dataBase, err := sql.Open("sqlite3", "/Users/achernov/job_db.bin")

	if err != nil {
		return dataBaseHandler, err
	}

	dataBaseMap := &gorp.DbMap{Db: dataBase, Dialect: gorp.SqliteDialect{}}
	dataBaseMap.AddTableWithName(JobTable{}, "Jobs").SetKeys(false, "ID").SetVersionCol(GorpVersionColumn)
	//dataBaseMap.AddTableWithName(JobStateTable{}, "States").SetKeys(false, "ID").SetVersionCol(GorpVersionColumn)
	dataBaseMap.AddTableWithName(TaskTable{}, "Tasks").SetKeys(false, "ID").SetVersionCol(GorpVersionColumn)
	dataBaseMap.AddTableWithName(ServiceTable{}, "Services").SetKeys(false, "ID").SetVersionCol(GorpVersionColumn)
	dataBaseMap.AddTableWithName(IOPortTable{}, "IOPorts").SetVersionCol(GorpVersionColumn)
	err = dataBaseMap.CreateTablesIfNotExists()

	dataBaseHandler = Db{*dataBaseMap}

	return dataBaseHandler, err
}

// Jobs

func (d *Db) AddJob(job Job) error {
	storedJob := d.MapJSON2StoredJob(job)
	return d.Insert(&storedJob)
}

func (d *Db) RemoveJob(jobID JobID) error {
	_, err := d.Exec("DELETE FROM Tasks WHERE jobID=?", jobID)
	if err == nil {
		_, err = d.Exec("DELETE FROM Jobs WHERE ID=?", jobID)
	}
	return err
}

func (d *Db) RemoveTask(taskID string) error {
	_, err := d.Exec("DELETE FROM Tasks WHERE ID=?", taskID)
	return err
}

func (d *Db) MapStoredJob2JSON(jobID JobID, storedJob JobTable) (Job, error) {
	var job Job
	var jobState JobState
	var linkedTasks []Task
	var storedTasks []TaskTable
	_, err := d.Select(&storedTasks, "SELECT * FROM Tasks WHERE JobID=?", string(jobID))
	if err == nil {
		for _, t := range storedTasks {
			var curTask Task
			curTask.Error = t.Error
			curTask.ConsoleOutput = t.ConsoleOutput
			curTask.ContainerID = dckr.ContainerID(t.ContainerID)
			curTask.ExitCode = t.ExitCode
			curTask.ID = t.ID
			curTask.Name = t.Name

			linkedTasks = append(linkedTasks, curTask)
		}
	}


	jobState.Error = storedJob.Error
	jobState.Status = storedJob.Status
	jobState.Code = storedJob.Code

	job.ID = JobID(storedJob.ID)
	job.ServiceID = ServiceID(storedJob.ServiceID)
	job.Input = storedJob.Input
	job.Created = storedJob.Created
	job.State = &jobState
	job.InputVolume = VolumeID(storedJob.InputVolume)
	job.OutputVolume = VolumeID(storedJob.OutputVolume)
	job.Tasks = linkedTasks
	return job, err
}

func (d *Db) MapJSON2StoredJob(job Job) JobTable {
	var storedJob JobTable
	storedJob.ID = string(job.ID)
	storedJob.ServiceID = string(job.ServiceID)
	storedJob.Input = job.Input
	storedJob.Created = job.Created
	storedJob.Error = job.State.Error
	storedJob.Status = job.State.Status
	storedJob.Code = job.State.Code
	storedJob.InputVolume = string(job.InputVolume)
	storedJob.OutputVolume = string(job.OutputVolume)
	return storedJob
}

// ListJobs exported
func (d *Db) ListJobs() ([]Job, error) {
	var jobs []Job
	var jobsFromTable []JobTable
	_, err := d.Select(&jobsFromTable, "SELECT * FROM Jobs ORDER BY ID")
	if err == nil {
		for _, j := range jobsFromTable {
			var curJob Job
			curJob, err = d.MapStoredJob2JSON(JobID(j.ID), j)
			if err == nil {
				jobs = append(jobs, curJob)
			} else {
				break
			}
		}

	}
	return jobs, err
}

func (d *Db) GetJob(jobID JobID) (Job, error) {
	var job Job
	var jobFromTable JobTable
	err := d.SelectOne(&jobFromTable, "SELECT * FROM jobs WHERE ID=?", jobID)
	if err == nil {
		job, err = d.MapStoredJob2JSON(JobID(jobFromTable.ID), jobFromTable)
	}

	return job, err
}

func (d *Db) SetJobState(jobID JobID, state JobState) error {
	var storedJob JobTable
	err := d.SelectOne(&storedJob, "SELECT * FROM jobs WHERE ID=?", jobID)
	if err == nil {
		storedJob.Error = state.Error
		storedJob.Status = state.Status
		storedJob.Code = state.Code
		_, err = d.Update(&storedJob)
	}
	return err
}

func (d *Db) SetJobInputVolume(jobID JobID, inputVolume VolumeID) error {
	var storedJob JobTable
	err := d.SelectOne(&storedJob, "SELECT * FROM jobs WHERE ID=?", jobID)
	if err == nil {
		storedJob.InputVolume = string(inputVolume)
		_, err = d.Update(&storedJob)
	}
	return err
}

func (d *Db) SetJobOutputVolume(jobID JobID, outputVolume VolumeID) error {
	var storedJob JobTable
	err := d.SelectOne(&storedJob, "SELECT * from jobs WHERE ID=?", jobID)
	if err == nil {
		storedJob.OutputVolume = string(outputVolume)
		_, err = d.Update(&storedJob)
	}
	return err
}

func (d *Db) AddJobTask(jobID JobID, taskName string, taskContainer dckr.ContainerID, taskError string, taskExitCode int, taskConsoleOutput *bytes.Buffer) error {
	fmt.Println("Adding task")
	var newTask TaskTable
	newTask.ID = uuid.New()
	newTask.Name = taskName
	newTask.ContainerID = string(taskContainer)
	newTask.Error = taskError
	newTask.ExitCode = taskExitCode
	newTask.ConsoleOutput = taskConsoleOutput.String()
	newTask.JobID = string(jobID)
	err := d.Insert(&newTask)

	fmt.Println(err)
	return err
}

// Services

func (d *Db) MapStoredService2JSON(serviceID ServiceID, storedService ServiceTable) (Service, error) {
	var service Service
	var storedInputPorts []IOPortTable
	var storedOutputPorts []IOPortTable
	var inputPorts []IOPort
	var outputPorts []IOPort

	_, err := d.Select(&storedInputPorts, "SELECT * FROM IOPorts WHERE IsInput=1 AND ServiceID=?", serviceID)
	if err == nil {
		for _, i := range storedInputPorts {
			var curInput IOPort
			curInput.ID = i.ID
			curInput.Name = i.Name
			curInput.Path = i.Path

			inputPorts = append(inputPorts, curInput)
		}
	}
	_, err = d.Select(&storedOutputPorts, "SELECT * FROM IOPorts WHERE IsInput=0 AND ServiceID=?", serviceID)
	if err == nil {
		for _, o := range storedOutputPorts {
			var curOutput IOPort
			curOutput.ID = o.ID
			curOutput.Name = o.Name
			curOutput.Path = o.Path

			outputPorts = append(outputPorts, curOutput)
		}
	}

	service.ID = ServiceID(storedService.ID)
	service.ImageID = dckr.ImageID(storedService.ImageID)
	service.Name = storedService.Name
	service.RepoTag = storedService.RepoTag
	service.Description = storedService.Description
	service.Version = storedService.Version
	service.Created = storedService.Created
	service.Size = storedService.Size
	service.Input = inputPorts
	service.Input = inputPorts
	service.Output = outputPorts

	return service, err
}

func (d *Db) MapJSON2StoredService(service Service) ServiceTable {
	var storedService ServiceTable
	storedService.ID = string(service.ID)
	storedService.ImageID = string(service.ImageID)
	storedService.Name = service.Name
	storedService.RepoTag = service.RepoTag
	storedService.Description = service.Description
	storedService.Version = service.Version
	storedService.Created = service.Created
	storedService.Size = service.Size
	return storedService
}

func (d *Db) AddIOPort(service Service) error {
	var err error
	for _, p := range service.Input {
		var curInputPort IOPortTable
		curInputPort.Path = p.Path
		curInputPort.Name = p.Name
		curInputPort.ID = p.ID
		curInputPort.IsInput = true
		curInputPort.ServiceID = string(service.ID)
		err = d.Insert(&curInputPort)
		if err != nil {
			break
		}
	}
	if err == nil {
		for _, p := range service.Output {
			var curOutputPort IOPortTable
			curOutputPort.Path = p.Path
			curOutputPort.Name = p.Name
			curOutputPort.ID = p.ID
			curOutputPort.IsInput = false
			curOutputPort.ServiceID = string(service.ID)
			err = d.Insert(&curOutputPort)
			if err != nil {
				break
			}
		}
	}

	return err
}

func (d *Db) AddService(service Service) error {
	storedService := d.MapJSON2StoredService(service)
	err := d.AddIOPort(service)
	if err == nil {
		err = d.Insert(&storedService)
	}

	return err
}

func (d *Db) ListServices() ([]Service, error) {
	var services []Service
	var servicesFromTable []ServiceTable
	_, err := d.Select(&servicesFromTable, "SELECT * FROM services ORDER BY ID")
	if err == nil {
		for _, s := range servicesFromTable {
			var curService Service
			curService, err = d.MapStoredService2JSON(ServiceID(s.ID), s)
			if err == nil {
				services = append(services, curService)
			} else {
				break
			}
		}

	}

	return services, err
}

func (d *Db) GetService(serviceID ServiceID) (Service, error) {
	var service Service
	var serviceFromTable ServiceTable
	err := d.SelectOne(&serviceFromTable, "SELECT * FROM services WHERE ID=?", serviceID)

	if err == nil {
		service, err = d.MapStoredService2JSON(ServiceID(serviceFromTable.ID), serviceFromTable)
	}

	return service, err
}

func (d *Db) NewServiceFromImage(image dckr.Image) Service {
	srv := Service{
		ID:      ServiceID(uuid.New()),
		ImageID: image.ID,
		RepoTag: image.RepoTag,
		Created: image.Created,
		Size:    image.Size,
	}

	for k, v := range image.Labels {
		if !strings.HasPrefix(k, GefSrvLabelPrefix) {
			continue
		}
		k = k[len(GefSrvLabelPrefix):]
		ks := strings.Split(k, ".")
		if len(ks) == 0 {
			continue
		}
		switch ks[0] {
		case "name":
			srv.Name = v
		case "description":
			srv.Description = v
		case "version":
			srv.Version = v
		case "input":
			addVecValue(&srv.Input, ks[1:], v)
		case "output":
			addVecValue(&srv.Output, ks[1:], v)
		default:
			log.Println("Unknown GEF service label: ", k, "=", v)
		}
	}

	{
		in := make([]IOPort, 0, len(srv.Input))
		for _, p := range srv.Input {
			if p.Path != "" {
				p.ID = fmt.Sprintf("input%d", len(in))
				in = append(in, p)
			}
		}
		srv.Input = in
	}
	{
		out := make([]IOPort, 0, len(srv.Output))
		for _, p := range srv.Output {
			if p.Path != "" {
				p.ID = fmt.Sprintf("output%d", len(out))
				out = append(out, p)
			}
		}
		srv.Output = out
	}

	return srv
}

func addVecValue(vec *[]IOPort, ks []string, value string) {
	if len(ks) < 2 {
		log.Println("ERROR: GEF service label I/O key error (need 'port number . key name')", ks)
		return
	}
	id, err := strconv.ParseUint(ks[0], 10, 8)
	if err != nil {
		log.Println("ERROR: GEF service label: expecting integer argument for IOPort, instead got: ", ks)
	}
	for len(*vec) < int(id)+1 {
		*vec = append(*vec, IOPort{})
	}
	switch ks[1] {
	case "name":
		(*vec)[id].Name = value
	case "path":
		(*vec)[id].Path = value
	}
}
