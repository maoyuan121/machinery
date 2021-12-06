package machinery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/RichardKnop/machinery/v2/backends/result"
	"github.com/RichardKnop/machinery/v2/config"
	"github.com/RichardKnop/machinery/v2/log"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/RichardKnop/machinery/v2/tracing"
	"github.com/RichardKnop/machinery/v2/utils"

	backendsiface "github.com/RichardKnop/machinery/v2/backends/iface"
	brokersiface "github.com/RichardKnop/machinery/v2/brokers/iface"
	lockiface "github.com/RichardKnop/machinery/v2/locks/iface"
	opentracing "github.com/opentracing/opentracing-go"
)

// Server is the main Machinery object and stores all configuration
// Server 是主要的 Machinery 对象，存储所有的配置
// 所有 task worker 进程都在 Server 上注册
type Server struct {
	config            *config.Config
	registeredTasks   *sync.Map
	broker            brokersiface.Broker
	backend           backendsiface.Backend
	lock              lockiface.Lock
	scheduler         *cron.Cron
	prePublishHandler func(*tasks.Signature)
}

// NewServer 创建一个 Server 实例
func NewServer(cnf *config.Config, brokerServer brokersiface.Broker, backendServer backendsiface.Backend, lock lockiface.Lock) *Server {
	srv := &Server{
		config:          cnf,
		registeredTasks: new(sync.Map),
		broker:          brokerServer,
		backend:         backendServer,
		lock:            lock,
		scheduler:       cron.New(),
	}

	// 运行 scheduler job
	go srv.scheduler.Run()

	return srv
}

// NewWorker 创建一个 Worker 实例
func (server *Server) NewWorker(consumerTag string, concurrency int) *Worker {
	return &Worker{
		server:      server,
		ConsumerTag: consumerTag,
		Concurrency: concurrency,
		Queue:       "",
	}
}

// NewCustomQueueWorker 使用自定义队列创建一个 Worker 实例
func (server *Server) NewCustomQueueWorker(consumerTag string, concurrency int, queue string) *Worker {
	return &Worker{
		server:      server,
		ConsumerTag: consumerTag,
		Concurrency: concurrency,
		Queue:       queue,
	}
}

// GetBroker 返回 broker
func (server *Server) GetBroker() brokersiface.Broker {
	return server.broker
}

// SetBroker 设置 broker
func (server *Server) SetBroker(broker brokersiface.Broker) {
	server.broker = broker
}

// GetBackend 返回 backend
func (server *Server) GetBackend() backendsiface.Backend {
	return server.backend
}

// SetBackend 设置 backend
func (server *Server) SetBackend(backend backendsiface.Backend) {
	server.backend = backend
}

// GetConfig 返回一个 config
func (server *Server) GetConfig() *config.Config {
	return server.config
}

// SetConfig 设置 config
func (server *Server) SetConfig(cnf *config.Config) {
	server.config = cnf
}

// SetPreTaskHandler 设置每个 publish handler（在发布前触发）
func (server *Server) SetPreTaskHandler(handler func(*tasks.Signature)) {
	server.prePublishHandler = handler
}

// RegisterTasks 一次注册所有的 task
func (server *Server) RegisterTasks(namedTaskFuncs map[string]interface{}) error {
	for _, task := range namedTaskFuncs {
		if err := tasks.ValidateTask(task); err != nil {
			return err
		}
	}
	for k, v := range namedTaskFuncs {
		server.registeredTasks.Store(k, v)
	}
	server.broker.SetRegisteredTaskNames(server.GetRegisteredTaskNames())
	return nil
}

// RegisterTask 注册一个 task
func (server *Server) RegisterTask(name string, taskFunc interface{}) error {
	if err := tasks.ValidateTask(taskFunc); err != nil {
		return err
	}
	server.registeredTasks.Store(name, taskFunc)
	server.broker.SetRegisteredTaskNames(server.GetRegisteredTaskNames())
	return nil
}

// IsTaskRegistered 如果一个 task 名已经注册过了，返回 true
func (server *Server) IsTaskRegistered(name string) bool {
	_, ok := server.registeredTasks.Load(name)
	return ok
}

// GetRegisteredTask 根据名字返回已注册的  task
func (server *Server) GetRegisteredTask(name string) (interface{}, error) {
	taskFunc, ok := server.registeredTasks.Load(name)
	if !ok {
		return nil, fmt.Errorf("Task not registered error: %s", name)
	}
	return taskFunc, nil
}

// SendTaskWithContext 将在发布之前注入跟踪上下文到 signature header
func (server *Server) SendTaskWithContext(ctx context.Context, signature *tasks.Signature) (*result.AsyncResult, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "SendTask", tracing.ProducerOption(), tracing.MachineryTag)
	defer span.Finish()

	// tag the span with some info about the signature
	signature.Headers = tracing.HeadersWithSpan(signature.Headers, span)

	// Make sure result backend is defined
	if server.backend == nil {
		return nil, errors.New("Result backend required")
	}

	// 如果没有设置 UUID 那么自动生成一个
	if signature.UUID == "" {
		taskID := uuid.New().String()
		signature.UUID = fmt.Sprintf("task_%v", taskID)
	}

	// 设置初始 task 状态为 PENDING
	if err := server.backend.SetStatePending(signature); err != nil {
		return nil, fmt.Errorf("Set state pending error: %s", err)
	}

	if server.prePublishHandler != nil {
		server.prePublishHandler(signature)
	}

	if err := server.broker.Publish(ctx, signature); err != nil {
		return nil, fmt.Errorf("Publish message error: %s", err)
	}

	return result.NewAsyncResult(signature, server.backend), nil
}

// SendTask 发布一个 task 到默认的队列
func (server *Server) SendTask(signature *tasks.Signature) (*result.AsyncResult, error) {
	return server.SendTaskWithContext(context.Background(), signature)
}

// SendChainWithContext 在发布前会把 trace context 注入到所有的 signature header 上
func (server *Server) SendChainWithContext(ctx context.Context, chain *tasks.Chain) (*result.ChainAsyncResult, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "SendChain", tracing.ProducerOption(), tracing.MachineryTag, tracing.WorkflowChainTag)
	defer span.Finish()

	tracing.AnnotateSpanWithChainInfo(span, chain)

	return server.SendChain(chain)
}

// SendChain 触发一系列的任务
func (server *Server) SendChain(chain *tasks.Chain) (*result.ChainAsyncResult, error) {
	_, err := server.SendTask(chain.Tasks[0])
	if err != nil {
		return nil, err
	}

	return result.NewChainAsyncResult(chain.Tasks, server.backend), nil
}

// SendGroupWithContext 在发布前会把 trace context 注入到所有的 signature header 上
func (server *Server) SendGroupWithContext(ctx context.Context, group *tasks.Group, sendConcurrency int) ([]*result.AsyncResult, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "SendGroup", tracing.ProducerOption(), tracing.MachineryTag, tracing.WorkflowGroupTag)
	defer span.Finish()

	tracing.AnnotateSpanWithGroupInfo(span, group, sendConcurrency)

	// Make sure result backend is defined
	if server.backend == nil {
		return nil, errors.New("Result backend required")
	}

	asyncResults := make([]*result.AsyncResult, len(group.Tasks))

	var wg sync.WaitGroup
	wg.Add(len(group.Tasks))
	errorsChan := make(chan error, len(group.Tasks)*2)

	// Init group
	server.backend.InitGroup(group.GroupUUID, group.GetUUIDs())

	// Init the tasks Pending state first
	for _, signature := range group.Tasks {
		if err := server.backend.SetStatePending(signature); err != nil {
			errorsChan <- err
			continue
		}
	}

	pool := make(chan struct{}, sendConcurrency)
	go func() {
		for i := 0; i < sendConcurrency; i++ {
			pool <- struct{}{}
		}
	}()

	for i, signature := range group.Tasks {

		if sendConcurrency > 0 {
			<-pool
		}

		go func(s *tasks.Signature, index int) {
			defer wg.Done()

			// Publish task

			err := server.broker.Publish(ctx, s)

			if sendConcurrency > 0 {
				pool <- struct{}{}
			}

			if err != nil {
				errorsChan <- fmt.Errorf("Publish message error: %s", err)
				return
			}

			asyncResults[index] = result.NewAsyncResult(s, server.backend)
		}(signature, i)
	}

	done := make(chan int)
	go func() {
		wg.Wait()
		done <- 1
	}()

	select {
	case err := <-errorsChan:
		return asyncResults, err
	case <-done:
		return asyncResults, nil
	}
}

// SendGroup 触发一组并发的 task
func (server *Server) SendGroup(group *tasks.Group, sendConcurrency int) ([]*result.AsyncResult, error) {
	return server.SendGroupWithContext(context.Background(), group, sendConcurrency)
}

// SendChordWithContext 在发布前会把 trace context 注入到所有的 signature header 上
func (server *Server) SendChordWithContext(ctx context.Context, chord *tasks.Chord, sendConcurrency int) (*result.ChordAsyncResult, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "SendChord", tracing.ProducerOption(), tracing.MachineryTag, tracing.WorkflowChordTag)
	defer span.Finish()

	tracing.AnnotateSpanWithChordInfo(span, chord, sendConcurrency)

	_, err := server.SendGroupWithContext(ctx, chord.Group, sendConcurrency)
	if err != nil {
		return nil, err
	}

	return result.NewChordAsyncResult(
		chord.Group.Tasks,
		chord.Callback,
		server.backend,
	), nil
}

// SendChord 使用回调触发一组并行任务
func (server *Server) SendChord(chord *tasks.Chord, sendConcurrency int) (*result.ChordAsyncResult, error) {
	return server.SendChordWithContext(context.Background(), chord, sendConcurrency)
}

// GetRegisteredTaskNames 返回已经注册的任务名切片
func (server *Server) GetRegisteredTaskNames() []string {
	taskNames := make([]string, 0)

	server.registeredTasks.Range(func(key, value interface{}) bool {
		taskNames = append(taskNames, key.(string))
		return true
	})
	return taskNames
}

// RegisterPeriodicTask 注册一个定期触发的任务
func (server *Server) RegisterPeriodicTask(spec, name string, signature *tasks.Signature) error {
	//check spec
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return err
	}

	f := func() {
		//get lock
		err := server.lock.LockWithRetries(utils.GetLockName(name, spec), schedule.Next(time.Now()).UnixNano()-1)
		if err != nil {
			return
		}

		//send task
		_, err = server.SendTask(tasks.CopySignature(signature))
		if err != nil {
			log.ERROR.Printf("periodic task failed. task name is: %s. error is %s", name, err.Error())
		}
	}

	_, err = server.scheduler.AddFunc(spec, f)
	return err
}

// RegisterPeriodicChain 注册一个定期触发的 chain
func (server *Server) RegisterPeriodicChain(spec, name string, signatures ...*tasks.Signature) error {
	//check spec
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return err
	}

	f := func() {
		// new chain
		chain, _ := tasks.NewChain(tasks.CopySignatures(signatures...)...)

		//get lock
		err := server.lock.LockWithRetries(utils.GetLockName(name, spec), schedule.Next(time.Now()).UnixNano()-1)
		if err != nil {
			return
		}

		//send task
		_, err = server.SendChain(chain)
		if err != nil {
			log.ERROR.Printf("periodic task failed. task name is: %s. error is %s", name, err.Error())
		}
	}

	_, err = server.scheduler.AddFunc(spec, f)
	return err
}

// RegisterPeriodicGroup 注册一个定期触发的 group
func (server *Server) RegisterPeriodicGroup(spec, name string, sendConcurrency int, signatures ...*tasks.Signature) error {
	//check spec
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return err
	}

	f := func() {
		// new group
		group, _ := tasks.NewGroup(tasks.CopySignatures(signatures...)...)

		//get lock
		err := server.lock.LockWithRetries(utils.GetLockName(name, spec), schedule.Next(time.Now()).UnixNano()-1)
		if err != nil {
			return
		}

		//send task
		_, err = server.SendGroup(group, sendConcurrency)
		if err != nil {
			log.ERROR.Printf("periodic task failed. task name is: %s. error is %s", name, err.Error())
		}
	}

	_, err = server.scheduler.AddFunc(spec, f)
	return err
}

// RegisterPeriodicChord 注册一个定期触发的 chord
func (server *Server) RegisterPeriodicChord(spec, name string, sendConcurrency int, callback *tasks.Signature, signatures ...*tasks.Signature) error {
	//check spec
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return err
	}

	f := func() {
		// new chord
		group, _ := tasks.NewGroup(tasks.CopySignatures(signatures...)...)
		chord, _ := tasks.NewChord(group, tasks.CopySignature(callback))

		//get lock
		err := server.lock.LockWithRetries(utils.GetLockName(name, spec), schedule.Next(time.Now()).UnixNano()-1)
		if err != nil {
			return
		}

		//send task
		_, err = server.SendChord(chord, sendConcurrency)
		if err != nil {
			log.ERROR.Printf("periodic task failed. task name is: %s. error is %s", name, err.Error())
		}
	}

	_, err = server.scheduler.AddFunc(spec, f)
	return err
}
