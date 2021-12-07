package tasks

import "time"

const (
	// StatePending - 任务的初始状态
	StatePending = "PENDING"
	// StateReceived - 当一个任务被 worker 接收了
	StateReceived = "RECEIVED"
	// StateStarted - 当 worker 开始处理这个任务
	StateStarted = "STARTED"
	// StateRetry - 当失败的任务被调度为重试
	StateRetry = "RETRY"
	// StateSuccess - 当任务被成功处理
	StateSuccess = "SUCCESS"
	// StateFailure - 任务处理失败
	StateFailure = "FAILURE"
)

// TaskState 标识一个任务的状态
type TaskState struct {
	TaskUUID  string        `bson:"_id"`
	TaskName  string        `bson:"task_name"`
	State     string        `bson:"state"`
	Results   []*TaskResult `bson:"results"`
	Error     string        `bson:"error"`
	CreatedAt time.Time     `bson:"created_at"`
	TTL       int64         `bson:"ttl,omitempty"`
}

// GroupMeta 存储同一组里面任务的 metadata
// E.g. UUIDs of all tasks which are used in order to check if all tasks
// completed successfully or not and thus whether to trigger chord callback
type GroupMeta struct {
	GroupUUID      string    `bson:"_id"`
	TaskUUIDs      []string  `bson:"task_uuids"`
	ChordTriggered bool      `bson:"chord_triggered"`
	Lock           bool      `bson:"lock"`
	CreatedAt      time.Time `bson:"created_at"`
	TTL            int64     `bson:"ttl,omitempty"`
}

// NewPendingTaskState ...
func NewPendingTaskState(signature *Signature) *TaskState {
	return &TaskState{
		TaskUUID:  signature.UUID,
		TaskName:  signature.Name,
		State:     StatePending,
		CreatedAt: time.Now().UTC(),
	}
}

// NewReceivedTaskState ...
func NewReceivedTaskState(signature *Signature) *TaskState {
	return &TaskState{
		TaskUUID: signature.UUID,
		State:    StateReceived,
	}
}

// NewStartedTaskState ...
func NewStartedTaskState(signature *Signature) *TaskState {
	return &TaskState{
		TaskUUID: signature.UUID,
		State:    StateStarted,
	}
}

// NewSuccessTaskState ...
func NewSuccessTaskState(signature *Signature, results []*TaskResult) *TaskState {
	return &TaskState{
		TaskUUID: signature.UUID,
		State:    StateSuccess,
		Results:  results,
	}
}

// NewFailureTaskState ...
func NewFailureTaskState(signature *Signature, err string) *TaskState {
	return &TaskState{
		TaskUUID: signature.UUID,
		State:    StateFailure,
		Error:    err,
	}
}

// NewRetryTaskState ...
func NewRetryTaskState(signature *Signature) *TaskState {
	return &TaskState{
		TaskUUID: signature.UUID,
		State:    StateRetry,
	}
}

// IsCompleted returns true if state is SUCCESS or FAILURE,
// i.e. the task has finished processing and either succeeded or failed.
func (taskState *TaskState) IsCompleted() bool {
	return taskState.IsSuccess() || taskState.IsFailure()
}

// IsSuccess returns true if state is SUCCESS
func (taskState *TaskState) IsSuccess() bool {
	return taskState.State == StateSuccess
}

// IsFailure returns true if state is FAILURE
func (taskState *TaskState) IsFailure() bool {
	return taskState.State == StateFailure
}
