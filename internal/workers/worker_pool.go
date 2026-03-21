package workers

import (
	"context"
	"sync"
	"time"

	"followupmedium-newsroom/internal/models"
	"followupmedium-newsroom/internal/services"

	"github.com/sirupsen/logrus"
)

type WorkerPool struct {
	size         int
	jobs         chan Job
	workers      []*Worker
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	storyService *services.StoryService
	diffEngine   *services.DiffEngine
}

type Job struct {
	Type    string
	StoryID string
	Source  models.Source
	Data    interface{}
}

type Worker struct {
	id           int
	jobs         <-chan Job
	storyService *services.StoryService
	diffEngine   *services.DiffEngine
	fetcher      *ContentFetcher
}

func NewWorkerPool(size int, storyService *services.StoryService, diffEngine *services.DiffEngine) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		size:         size,
		jobs:         make(chan Job, size*2), // Buffer for jobs
		workers:      make([]*Worker, size),
		ctx:          ctx,
		cancel:       cancel,
		storyService: storyService,
		diffEngine:   diffEngine,
	}
}

func (wp *WorkerPool) Start() {
	logrus.Infof("Starting worker pool with %d workers", wp.size)
	
	for i := 0; i < wp.size; i++ {
		worker := &Worker{
			id:           i,
			jobs:         wp.jobs,
			storyService: wp.storyService,
			diffEngine:   wp.diffEngine,
			fetcher:      NewContentFetcher(),
		}
		wp.workers[i] = worker
		
		wp.wg.Add(1)
		go worker.start(wp.ctx, &wp.wg)
	}

	// Start periodic story monitoring
	go wp.startPeriodicMonitoring()
}

func (wp *WorkerPool) Stop() {
	logrus.Info("Stopping worker pool...")
	wp.cancel()
	close(wp.jobs)
	wp.wg.Wait()
	logrus.Info("Worker pool stopped")
}

func (wp *WorkerPool) SubmitJob(job Job) {
	select {
	case wp.jobs <- job:
		logrus.WithFields(logrus.Fields{
			"job_type":  job.Type,
			"story_id":  job.StoryID,
			"source":    job.Source.Name,
		}).Debug("Job submitted to worker pool")
	case <-wp.ctx.Done():
		logrus.Warn("Cannot submit job: worker pool is shutting down")
	default:
		logrus.Warn("Job queue is full, dropping job")
	}
}

func (wp *WorkerPool) startPeriodicMonitoring() {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wp.monitorActiveStories()
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) monitorActiveStories() {
	stories, err := wp.storyService.GetActiveStories()
	if err != nil {
		logrus.WithError(err).Error("Failed to get active stories for monitoring")
		return
	}

	logrus.WithField("story_count", len(stories)).Debug("Monitoring active stories")

	for _, story := range stories {
		for _, source := range story.Sources {
			job := Job{
				Type:    "fetch_content",
				StoryID: story.ID.Hex(),
				Source:  source,
			}
			wp.SubmitJob(job)
		}
	}
}

func (w *Worker) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	
	logrus.WithField("worker_id", w.id).Info("Worker started")
	
	for {
		select {
		case job, ok := <-w.jobs:
			if !ok {
				logrus.WithField("worker_id", w.id).Info("Worker stopping: job channel closed")
				return
			}
			w.processJob(job)
		case <-ctx.Done():
			logrus.WithField("worker_id", w.id).Info("Worker stopping: context cancelled")
			return
		}
	}
}

func (w *Worker) processJob(job Job) {
	startTime := time.Now()
	
	logrus.WithFields(logrus.Fields{
		"worker_id": w.id,
		"job_type":  job.Type,
		"story_id":  job.StoryID,
	}).Debug("Processing job")

	switch job.Type {
	case "fetch_content":
		w.processFetchContent(job)
	default:
		logrus.WithField("job_type", job.Type).Warn("Unknown job type")
	}

	logrus.WithFields(logrus.Fields{
		"worker_id": w.id,
		"job_type":  job.Type,
		"latency":   time.Since(startTime).Milliseconds(),
	}).Debug("Job completed")
}

func (w *Worker) processFetchContent(job Job) {
	// Fetch content from source
	content, err := w.fetcher.FetchContent(job.Source)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"story_id": job.StoryID,
			"source":   job.Source.Name,
			"error":    err.Error(),
		}).Error("Failed to fetch content")
		return
	}

	if content == "" {
		logrus.WithFields(logrus.Fields{
			"story_id": job.StoryID,
			"source":   job.Source.Name,
		}).Debug("No new content found")
		return
	}

	// Compute diff
	diffResult, err := w.diffEngine.ComputeContentDiff(job.StoryID, content, job.Source)
	if err != nil {
		logrus.WithError(err).Error("Failed to compute content diff")
		return
	}

	if !diffResult.IsNew {
		logrus.WithField("story_id", job.StoryID).Debug("Content already processed")
		return
	}

	// Create development
	development := models.Development{
		Content:     content,
		Source:      job.Source,
		ContentHash: diffResult.ContentHash,
		Type:        diffResult.DiffType,
		Metadata:    make(map[string]string),
	}

	// Add development to story
	if err := w.storyService.AddDevelopment(job.StoryID, development); err != nil {
		logrus.WithError(err).Error("Failed to add development to story")
		return
	}

	logrus.WithFields(logrus.Fields{
		"story_id":        job.StoryID,
		"development_type": diffResult.DiffType,
		"source":          job.Source.Name,
	}).Info("New development added to story")
}