package iface

import (
	"github.com/RichardKnop/machinery/v2/tasks"
)

// Backend - 用于所有结果后端的公共接口
type Backend interface {
	// Group 相关的函数
	InitGroup(groupUUID string, taskUUIDs []string) error
	GroupCompleted(groupUUID string, groupTaskCount int) (bool, error)
	GroupTaskStates(groupUUID string, groupTaskCount int) ([]*tasks.TaskState, error)
	TriggerChord(groupUUID string) (bool, error)

	// 获取/设置任务状态
	SetStatePending(signature *tasks.Signature) error
	SetStateReceived(signature *tasks.Signature) error
	SetStateStarted(signature *tasks.Signature) error
	SetStateRetry(signature *tasks.Signature) error
	SetStateSuccess(signature *tasks.Signature, results []*tasks.TaskResult) error
	SetStateFailure(signature *tasks.Signature, err string) error
	GetState(taskUUID string) (*tasks.TaskState, error)

	// 清除存储的存储任务状态和组元数据
	IsAMQP() bool
	PurgeState(taskUUID string) error
	PurgeGroupMeta(groupUUID string) error
}
