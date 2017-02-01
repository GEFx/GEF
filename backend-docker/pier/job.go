package pier

import (
	"sort"
	"sync"
	"time"

	"github.com/EUDAT-GEF/GEF/backend-docker/pier/internal/dckr"
)

// Job stores the information about a service execution
type Job struct {
	ID           JobID
	ServiceID    ServiceID
	Input        string
	Created      time.Time
	State        *JobState
	OutputVolume VolumeID
}

// JobState export
type JobState struct {
	Error  error
	Status string
}

// JobID exported
type JobID string

func (job *Job) SetState(state JobState) {
	// TODO: set this atomically
	job.State = &state
}

type jobArray []Job

func (jl jobArray) Len() int {
	return len(jl)
}
func (jl jobArray) Swap(i, j int) {
	jl[i], jl[j] = jl[j], jl[i]
}
func (jl jobArray) Less(i, j int) bool {
	return jl[i].ID < jl[j].ID
}

// Bind describes the binding between an IOPort and a docker volume
type Bind struct {
	IOPort   IOPort
	VolumeID dckr.VolumeID
}

// JobList is a shared structure that stores info about all jobs
type JobList struct {
	sync.Mutex
	cache map[JobID]Job
}

// NewJobList exported
func NewJobList() *JobList {
	return &JobList{
		cache: make(map[JobID]Job),
	}
}

func (jobList *JobList) add(job Job) {
	jobList.Lock()
	defer jobList.Unlock()
	jobList.cache[job.ID] = job
}

func (jobList *JobList) list() []Job {
	jobList.Lock()
	defer jobList.Unlock()
	all := make([]Job, len(jobList.cache), len(jobList.cache))
	i := 0
	for _, job := range jobList.cache {
		all[i] = job
		i++
	}
	sort.Sort(jobArray(all))
	return all
}

func (jobList *JobList) get(key JobID) (Job, bool) {
	jobList.Lock()
	defer jobList.Unlock()
	job, ok := jobList.cache[key]
	return job, ok
}