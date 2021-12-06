[1]: https://raw.githubusercontent.com/RichardKnop/assets/master/machinery/example_worker.png
[2]: https://raw.githubusercontent.com/RichardKnop/assets/master/machinery/example_worker_receives_tasks.png
[3]: http://patreon_public_assets.s3.amazonaws.com/sized/becomeAPatronBanner.png

## Machinery

Machinery 是一种基于分布式消息传递的异步任务队列/作业队列。

[![Travis Status for RichardKnop/machinery](https://travis-ci.org/RichardKnop/machinery.svg?branch=master&label=linux+build)](https://travis-ci.org/RichardKnop/machinery)
[![godoc for RichardKnop/machinery](https://godoc.org/github.com/nathany/looper?status.svg)](http://godoc.org/github.com/RichardKnop/machinery/v1)
[![codecov for RichardKnop/machinery](https://codecov.io/gh/RichardKnop/machinery/branch/master/graph/badge.svg)](https://codecov.io/gh/RichardKnop/machinery)

[![Go Report Card](https://goreportcard.com/badge/github.com/RichardKnop/machinery)](https://goreportcard.com/report/github.com/RichardKnop/machinery)
[![GolangCI](https://golangci.com/badges/github.com/RichardKnop/machinery.svg)](https://golangci.com)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)

[![Sourcegraph for RichardKnop/machinery](https://sourcegraph.com/github.com/RichardKnop/machinery/-/badge.svg)](https://sourcegraph.com/github.com/RichardKnop/machinery?badge)
[![Donate Bitcoin](https://img.shields.io/badge/donate-bitcoin-orange.svg)](https://richardknop.github.io/donate/)

---

* [V2 Experiment](#v2-experiment)
* [First Steps](#first-steps)
* [Configuration](#configuration)
  * [Lock](#lock)
  * [Broker](#broker)
  * [DefaultQueue](#defaultqueue)
  * [ResultBackend](#resultbackend)
  * [ResultsExpireIn](#resultsexpirein)
  * [AMQP](#amqp-2)
  * [DynamoDB](#dynamodb)
  * [Redis](#redis-2)
  * [GCPPubSub](#gcppubsub)
* [Custom Logger](#custom-logger)
* [Server](#server)
* [Workers](#workers)
* [Tasks](#tasks)
  * [Registering Tasks](#registering-tasks)
  * [Signatures](#signatures)
  * [Supported Types](#supported-types)
  * [Sending Tasks](#sending-tasks)
  * [Delayed Tasks](#delayed-tasks)
  * [Retry Tasks](#retry-tasks)
  * [Get Pending Tasks](#get-pending-tasks)
  * [Keeping Results](#keeping-results)
* [Workflows](#workflows)
  * [Groups](#groups)
  * [Chords](#chords)
  * [Chains](#chains)
* [Periodic Tasks & Workflows](#periodic-tasks--workflows)
  * [Periodic Tasks](#periodic-tasks)
  * [Periodic Groups](#periodic-groups)
  * [Periodic Chains](#periodic-chains)
  * [Periodic Chords](#periodic-chords)
* [Development](#development)
  * [Requirements](#requirements)
  * [Dependencies](#dependencies)
  * [Testing](#testing)

### V2 Experiment

请注意，V2 的工作正在进行中，在准备就绪之前，可能也将发生破坏性的更改。

您可以使用当前的 V2，以避免必须导入未使用的代理和后端的所有依赖项。

不是工厂，你需要注入代理和后端对象到服务器构造函数:

```go
import (
  "github.com/RichardKnop/machinery/v2"
  backendsiface "github.com/RichardKnop/machinery/v2/backends/iface"
  brokersiface "github.com/RichardKnop/machinery/v2/brokers/iface"
  locksiface "github.com/RichardKnop/machinery/v2/locks/iface"
)

var broker brokersiface.Broker
var backend backendsiface.Backend
var lock locksiface.Lock
server := machinery.NewServer(cnf, broker, backend, lock)
// server.NewWorker("machinery", 10)
```

### 第一步

将 Machinery 库添加到 $GOPATH/src：

```sh
go get github.com/RichardKnop/machinery/v1
```

或获得实验性 v2 版本：

```sh
go get github.com/RichardKnop/machinery/v2
```


首先，您需要定义一些任务。查看 `example/tasks/tasks` 中的示例任务。去看看几个例子吧。

其次，你需要使用以下命令之一来启动一个 worker 进程（推荐使用 v2，因为它不会为所有的 broker / backend 导入依赖，只导入那些你真正需要的依赖）:


```sh
go run example/amqp/main.go worker
go run example/redis/main.go worker

go run example/amqp/main.go worker
go run example/redis/main.go worker
```

您也可以尝试 v2 示例。

```sh
cd v2/
go run example/amqp/main.go worker
go run example/redigo/main.go worker // Redis with redigo driver
go run example/go-redis/main.go worker // Redis with Go Redis driver

go run example/amqp/main.go worker
go run example/redis/main.go worker
```

![Example worker][1]

最后，当你有一个 worker 在运行并等待任务消耗时，用以下命令发送一些任务（推荐 v2，因为它不为所有的broker / backend导入依赖，只导入那些你真正需要的依赖）:



```sh
go run example/v2/amqp/main.go send
go run example/v2/redigo/main.go send // Redis with redigo driver
go run example/v2/go-redis/main.go send // Redis with Go Redis driver

go run example/v1/amqp/main.go send
go run example/v1/redis/main.go send
```


你将能够看到工作线程异步地处理任务:

![Example worker receives tasks][2]

### Configuration

[config](/v1/config/config.go) 包提供了从环境变量或 YAML 文件加载配置的方便方法。例如，从环境变量中加载配置：

```go
cnf, err := config.NewFromEnvironment()
```

或者从 YAML 文件加载：

```go
cnf, err := config.NewFromYaml("config.yml", true)
```

第二个布尔标志允许每 10 秒重新加载一次配置。使用 `false` 禁用实时重新加载。

Machinery 配置由一个 `Config` 结构封装，并作为依赖注入到需要它的对象中。

#### Lock

##### Redis

使用Redis URL在以下格式之一:

```
redis://[password@]host[port][/db_num]
```

例如：

1. `redis://localhost:6379`, or with password `redis://password@localhost:6379`

#### Broker

一个消息 broker。目前支持的 broker 有：

##### AMQP

使用 AMQP URL 的格式：

```
amqp://[username:password@]@host[:port]
```

例如：
1. `amqp://guest:guest@localhost:5672`

AMQP 还支持多个代理 url。你需要在 `MultipleBrokerSeparator` 字段中指定 URL 分隔符。

##### Redis

使用 Redis URL 的格式：

```
redis://[password@]host[port][/db_num]
redis+socket://[password@]/path/to/file.sock[:/db_num]
```

例如：

1. `redis://localhost:6379`, or with password `redis://password@localhost:6379`
2. `redis+socket://password@/path/to/file.sock:/0`

##### AWS SQS

使用 AWS SQS URL 的格式：

```
https://sqs.us-east-2.amazonaws.com/123456789012
```

更多信息请查看 [AWS SQS docs](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html)。 
此外，还需要配置 `AWS_REGION`，否则将抛出错误。

要使用手动配置的 SQS 客户端：

```go
var sqsClient = sqs.New(session.Must(session.NewSession(&aws.Config{
  Region:         aws.String("YOUR_AWS_REGION"),
  Credentials:    credentials.NewStaticCredentials("YOUR_AWS_ACCESS_KEY", "YOUR_AWS_ACCESS_SECRET", ""),
  HTTPClient:     &http.Client{
    Timeout: time.Second * 120,
  },
})))
var visibilityTimeout = 20
var cnf = &config.Config{
  Broker:          "YOUR_SQS_URL"
  DefaultQueue:    "machinery_tasks",
  ResultBackend:   "YOUR_BACKEND_URL",
  SQS: &config.SQSConfig{
    Client: sqsClient,
    // if VisibilityTimeout is nil default to the overall visibility timeout setting for the queue
    // https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html
    VisibilityTimeout: &visibilityTimeout,
    WaitTimeSeconds: 30,
  },
}
```

##### GCP Pub/Sub

使用 GCP Pub/Sub URL 的格式：

```
gcppubsub://YOUR_GCP_PROJECT_ID/YOUR_PUBSUB_SUBSCRIPTION_NAME
```

使用手动配置的发布/订阅客户端:

```go
pubsubClient, err := pubsub.NewClient(
    context.Background(),
    "YOUR_GCP_PROJECT_ID",
    option.WithServiceAccountFile("YOUR_GCP_SERVICE_ACCOUNT_FILE"),
)

cnf := &config.Config{
  Broker:          "gcppubsub://YOUR_GCP_PROJECT_ID/YOUR_PUBSUB_SUBSCRIPTION_NAME"
  DefaultQueue:    "YOUR_PUBSUB_TOPIC_NAME",
  ResultBackend:   "YOUR_BACKEND_URL",
  GCPPubSub: config.GCPPubSubConfig{
    Client: pubsubClient,
  },
}
```

#### DefaultQueue

默认队列名，例如： `machinery_tasks`。

#### ResultBackend

用于保存任务状态和结果的结果后端。

目前支持的后端为：

##### Redis

使用下面的 Redis URL 格式之一：

```
redis://[password@]host[port][/db_num]
redis+socket://[password@]/path/to/file.sock[:/db_num]
```

例子：

1. `redis://localhost:6379`, or with password `redis://password@localhost:6379`
2. `redis+socket://password@/path/to/file.sock:/0`
3. cluster `redis://host1:port1,host2:port2,host3:port3`
4. cluster with password `redis://pass@host1:port1,host2:port2,host3:port3`

##### Memcache

使用 Memcache URL 的格式：

```
memcache://host1[:port1][,host2[:port2],...[,hostN[:portN]]]
```

例子：

1. `memcache://localhost:11211` for a single instance, or
2. `memcache://10.0.0.1:11211,10.0.0.2:11211` for a cluster

##### AMQP

使用 AMQP URL 的格式：

```
amqp://[username:password@]@host[:port]
```

例子：

1. `amqp://guest:guest@localhost:5672`

> 记住不推荐使用 AMQP 作为 result backend。查看 [Keeping Results](https://github.com/RichardKnop/machinery#keeping-results)

##### MongoDB

使用 MongoDB URL 的格式：

```
mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
```

例子：

1. `mongodb://localhost:27017/taskresults`

更多信息请查看 [MongoDB docs](https://docs.mongodb.org/manual/reference/connection-string/)。

#### ResultsExpireIn

将结果存储多久，单位为秒。默认为 `3600` （1 个小时）。

#### AMQP

RabbitMQ 相关的配置。如果你使用其它的  broker/backend 则不需要。

* `Exchange`: 交换机名，例如：`machinery_exchange`
* `ExchangeType`: 交换机类型，例如：`direct`
* `QueueBindingArguments`: 绑定到 AMQP 队列时使用的附加参数的可选映射
* `BindingKey`: The queue is bind to the exchange with this key, e.g. `machinery_task`
* `PrefetchCount`: How many tasks to prefetch (set to `1` if you have long running tasks)

#### DynamoDB

DynamoDB 相关的配置。如果你使用的是其它的 backend 那么不需要关心下面的东西。

* `TaskStatesTable`: 保存任务状态的自定义表名。 默认是 `task_states`，确保首先在您的 AWS 管理中创建这个表，使用 `TaskUUID` 作为表的主键。
* `GroupMetasTable`: 存组元数据的自定义表名。默认值是 `group_metas`，并确保首先在您的 AWS 管理中创建这个表，使用 `GroupUUID` 作为表的主键。

例子：

```
dynamodb:
  task_states_table: 'task_states'
  group_metas_table: 'group_metas'
```

如果找不到这些表，将抛出一个致命错误。

如果您希望使记录过期，您可以在 AWS admin 中为这些表配置 `TTL` 字段。TTL 字段是根据服务器配置中的 `ResultsExpireIn` 值设置的。更多信息请参见 https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html。


#### Redis

Redis 相关的配置。如果你使用的是其它的 backend 那么不需要关心下面的东西。

See: [config](/v1/config/config.go) (TODO)

#### GCPPubSub

GCPPubSub 相关的配置。如果你使用的是其它的 backend 那么不需要关心下面的东西。

See: [config](/v1/config/config.go) (TODO)

### Custom Logger

你可以通过实现下面的接口来自定义 logger：

```go
type Interface interface {
  Print(...interface{})
  Printf(string, ...interface{})
  Println(...interface{})

  Fatal(...interface{})
  Fatalf(string, ...interface{})
  Fatalln(...interface{})

  Panic(...interface{})
  Panicf(string, ...interface{})
  Panicln(...interface{})
}
```


然后只需在设置代码中通过调用 `github.com/RichardKnop/machinery/v1/log` 包导出的 `set` 函数来设置 logger:

```go
log.Set(myCustomLogger)
```

### Server

在使用之前，必须实例化 Machinery 库。这是通过创建一个 `Server` 实例来实现的。 `Server` 是存储 machine 配置和注册任务的基本对象。例如:

```go
import (
  "github.com/RichardKnop/machinery/v1/config"
  "github.com/RichardKnop/machinery/v1"
)

var cnf = &config.Config{
  Broker:        "amqp://guest:guest@localhost:5672/",
  DefaultQueue:  "machinery_tasks",
  ResultBackend: "amqp://guest:guest@localhost:5672/",
  AMQP: &config.AMQPConfig{
    Exchange:     "machinery_exchange",
    ExchangeType: "direct",
    BindingKey:   "machinery_task",
  },
}

server, err := machinery.NewServer(cnf)
if err != nil {
  // do something with the error
}
```

### Workers

为了使用任务，您需要运行一个或多个 worker。运行 worker 所需要的只是一个带有已注册任务的 `Server` 实例。例如:

```go
worker := server.NewWorker("worker_name", 10)
err := worker.Launch()
if err != nil {
  // do something with the error
}
```

每个 worker 将只使用已注册的任务。对于队列中的每个任务，Worker.Process() 方法将在 goroutine 中运行。
使用 `server.NewWorker` 的第二个参数来限制同时运行 Worker.Process() 调用的数量(每个 worker)。
示例：1 将序列化任务执行，而 0 将使并发执行的任务数量无限（默认）。

### Tasks

任务是 Machinery 应用程序的组成部分。task 是一个函数，它定义了当一个 worker 接收到一条消息时会发生什么。

每个任务需要返回一个错误作为最后一个返回值。除了错误之外，任务现在还可以返回任意数量的参数。

有效任务的例子:

```go
func Add(args ...int64) (int64, error) {
  sum := int64(0)
  for _, arg := range args {
    sum += arg
  }
  return sum, nil
}

func Multiply(args ...int64) (int64, error) {
  sum := int64(1)
  for _, arg := range args {
    sum *= arg
  }
  return sum, nil
}

// You can use context.Context as first argument to tasks, useful for open tracing
func TaskWithContext(ctx context.Context, arg Arg) error {
  // ... use ctx ...
  return nil
}

// Tasks need to return at least error as a minimal requirement
func DummyTask(arg string) error {
  return errors.New(arg)
}

// You can also return multiple results from the task
func DummyTask2(arg1, arg2 string) (string, string, error) {
  return arg1, arg2, nil
}
```

#### Registering Tasks

在您的 worker 使用任务之前，您需要将其注册到 server。这是通过给一个任务分配一个唯一的名称来实现的:

```go
server.RegisterTasks(map[string]interface{}{
  "add":      Add,
  "multiply": Multiply,
})
```

任务也可以一个接一个的注册：

```go
server.RegisterTask("add", Add)
server.RegisterTask("multiply", Multiply)
```

简单地说，当一个 worker 收到这样的信息:

```json
{
  "UUID": "48760a1a-8576-4536-973b-da09048c2ac5",
  "Name": "add",
  "RoutingKey": "",
  "ETA": null,
  "GroupUUID": "",
  "GroupTaskCount": 0,
  "Args": [
    {
      "Type": "int64",
      "Value": 1,
    },
    {
      "Type": "int64",
      "Value": 1,
    }
  ],
  "Immutable": false,
  "RetryCount": 0,
  "RetryTimeout": 0,
  "OnSuccess": null,
  "OnError": null,
  "ChordCallback": null
}
```

它将调用 Add(1,1)。每个任务也应该返回一个错误，以便我们可以处理失败。

理想情况下，任务应该是幂等的，这意味着当一个任务以相同的参数被多次调用时，不会出现意想不到的结果。

#### Signatures

一个签名包装了一个任务的调用参数、执行选项(例如不变性)和成功/错误回调，因此它可以通过网络发送给 worker。任务签名实现了一个简单的接口:

```go
// Arg 表示传递给任务调用的单个参数
type Arg struct {
  Type  string
  Value interface{}
}

// Header 表示用于指导任务的头文件
type Headers map[string]interface{}

// Signature 表示单个任务调用
type Signature struct {
  UUID           string
  Name           string
  RoutingKey     string
  ETA            *time.Time
  GroupUUID      string
  GroupTaskCount int
  Args           []Arg
  Headers        Headers
  Immutable      bool
  RetryCount     int
  RetryTimeout   int
  OnSuccess      []*Signature
  OnError        []*Signature
  ChordCallback  *Signature
}
```

`UUID` 是任务的唯一 id.你可以自己设置，也可以自动生成。

`Name` 是针对 Server 实例注册的惟一任务名称。

`RoutingKey` 用于路由任务到正确的队列。如果您让它为空，默认行为将是为 direct 交换类型将它设置为默认队列的绑定键，为其他交换类型设置为默认队列名称。

`ETA` 是用于延迟任务的时间戳。如果它为 nil，任务将被发布给 worker 立即使用。如果设置了该参数，任务将被延迟到 ETA 时间戳。

`GroupUUID`, `GroupTaskCount` 对于创建任务组很有用。                             

`Args` 是 worker 执行任务时将传递给该任务的参数列表。

`Headers` 是将任务发布到 AMQP 队列时使用的标头列表。

`Immutable` 是一个标志，它定义了执行任务的结果是否可以修改。这对于 `OnSuccess` 回调很重要。不可变任务不会将其结果传递给它的成功回调函数，而可变任务会将其结果前置到发送给回调任务的参数。
长话短说，如果你想把链中第一个任务的结果传递给第二个任务，就把 Immutable 设为 false。

`RetryCount` 指定一个失败的任务应该重试的次数（默认为 0）。重试尝试将在时间间隔内进行，在每次失败后，将在未来安排另一个尝试。

`RetryTimeout` 指定将任务重新发送到队列进行重试尝试之前要等待多长时间。默认行为是使用斐波那契序列来增加每次失败的重试尝试后的超时时间。

`OnSuccess` 定义任务成功执行后将调用的任务。它是一个任务签名结构的切片。

`OnError` 定义任务执行失败后将调用的任务。传递给错误回调函数的第一个参数是失败任务返回的错误字符串。

`ChordCallback` 用于创建对一组任务的回调。

#### Supported Types

Machinery 在将任务发送给代理之前将它们编码为 JSON。任务结果也作为 JSON 编码的字符串存储在后端。因此，只能支持带有原生 JSON 表示的类型。
目前支持的类型有：

* `bool`
* `int`
* `int8`
* `int16`
* `int32`
* `int64`
* `uint`
* `uint8`
* `uint16`
* `uint32`
* `uint64`
* `float32`
* `float64`
* `string`
* `[]bool`
* `[]int`
* `[]int8`
* `[]int16`
* `[]int32`
* `[]int64`
* `[]uint`
* `[]uint8`
* `[]uint16`
* `[]uint32`
* `[]uint64`
* `[]float32`
* `[]float64`
* `[]string`

#### Sending Tasks

可以通过 `Signature` 实例传递给 `Server` 实例来调用任务。例：

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
)

signature := &tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

asyncResult, err := server.SendTask(signature)
if err != nil {
  // failed to send the task
  // do something with the error
}
```

#### Delayed Tasks

你可以通过在任务签名上设置时间戳字段来延迟任务。

```go
// Delay the task by 5 seconds
eta := time.Now().UTC().Add(time.Second * 5)
signature.ETA = &eta
```

#### Retry Tasks

在将任务声明为失败之前，可以设置多个重试尝试。将使用斐波那契序列将重试请求随时间间隔。(详见 `RetryTimeout`。)

```go
// If the task fails, retry it up to 3 times
signature.RetryCount = 3
```

或者，你可以从你的任务中返回 `tasks.ErrRetryTaskLater`，并指定任务应该重试的持续时间，例如：



```go
return tasks.NewErrRetryTaskLater("some error", 4 * time.Hour)
```

#### Get Pending Tasks

当前在队列中等待被 worker 消耗的任务可以被检查，例如:

```go
server.GetBroker().GetPendingTasks("some_queue")
```

> 目前只支持 Redis broker。

#### Keeping Results

如果配置结果后端，任务状态和结果将被持久化。可能的状态:

```go
const (
	// StatePending - initial state of a task
	StatePending = "PENDING"
	// StateReceived - when task is received by a worker
	StateReceived = "RECEIVED"
	// StateStarted - when the worker starts processing the task
	StateStarted = "STARTED"
	// StateRetry - when failed task has been scheduled for retry
	StateRetry = "RETRY"
	// StateSuccess - when the task is processed successfully
	StateSuccess = "SUCCESS"
	// StateFailure - when processing of the task fails
	StateFailure = "FAILURE"
)
```

> 当使用 AMQP 作为结果后端时，任务状态将持久化在每个任务的单独队列中。虽然 RabbitMQ 可以扩展到数千个队列，但当你希望运行大量并行任务时，强烈建议使用一个更适合的结果后端（例如 Memcache）。

```go
// TaskResult 表示已处理任务的实际返回值
type TaskResult struct {
  Type  string      `bson:"type"`
  Value interface{} `bson:"value"`
}

// TaskState 标识一个任务的状态
type TaskState struct {
  TaskUUID  string        `bson:"_id"`
  State     string        `bson:"state"`
  Results   []*TaskResult `bson:"results"`
  Error     string        `bson:"error"`
}

// GroupMeta 存储关于同一组内任务的有用元数据
// E.g. 所有任务的 UUIDs，用于检查所有任务是否成功完成，从而是否触发回调
  
type GroupMeta struct {
  GroupUUID      string   `bson:"_id"`
  TaskUUIDs      []string `bson:"task_uuids"`
  ChordTriggered bool     `bson:"chord_triggered"`
  Lock           bool     `bson:"lock"`
}
```

`TaskResult` 表示已处理任务的返回值切片。

`TaskState` 结构将在每次任务状态更改时被序列化并存储。

`GroupMeta` 存储关于同一组内任务的有用元数据。例如，所有任务的 uuid，用于检查所有任务是否成功完成，从而是否触发回调。

`AsyncResult` 对象允许您检查任务的状态:

```go
taskState := asyncResult.GetState()
fmt.Printf("Current state of %v task is:\n", taskState.TaskUUID)
fmt.Println(taskState.State)
```

有几种方便的方法来检查任务状态:

```go
asyncResult.GetState().IsCompleted()
asyncResult.GetState().IsSuccess()
asyncResult.GetState().IsFailure()
```

你也可以做一个同步阻塞调用来等待一个任务结果：

```go
results, err := asyncResult.Get(time.Duration(time.Millisecond * 5))
if err != nil {
  // getting result of a task failed
  // do something with the error
}
for _, result := range results {
  fmt.Println(result.Interface())
}
```

#### Error Handling

当一个任务返回一个错误时，默认的行为是首先尝试重试该任务，如果它是可重试的，否则记录错误，然后最终调用任何错误回调。

为了自定义这一点，你可以在 worker 上设置一个自定义的错误处理程序，它可以做的不仅仅是在重试失败和错误回调触发后记录日志:

```go
worker.SetErrorHandler(func (err error) {
  customHandler(err)
})
```

### Workflows


运行单个异步任务很好，但通常您会希望设计一个任务工作流，以一种精心编排的方式执行。有几个有用的函数可以帮助您设计工作流。

#### Groups

`Group` 是一组将并行执行的任务，它们彼此独立。例如:

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

group, _ := tasks.NewGroup(&signature1, &signature2)
asyncResults, err := server.SendGroup(group, 0) //The second parameter specifies the number of concurrent sending tasks. 0 means unlimited.
if err != nil {
  // failed to send the group
  // do something with the error
}
```

`SendGroup` 返回一个 `AsyncResult` 对象的切片。所以你可以做一个阻塞调用，并等待组任务的结果:

```go
for _, asyncResult := range asyncResults {
  results, err := asyncResult.Get(time.Duration(time.Millisecond * 5))
  if err != nil {
    // getting result of a task failed
    // do something with the error
  }
  for _, result := range results {
    fmt.Println(result.Interface())
  }
}
```

#### Chords

`Chord` 允许你定义一个回调函数，在一个组的所有任务完成后执行，例如:

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

signature3 := tasks.Signature{
  Name: "multiply",
}

group := tasks.NewGroup(&signature1, &signature2)
chord, _ := tasks.NewChord(group, &signature3)
chordAsyncResult, err := server.SendChord(chord, 0) //The second parameter specifies the number of concurrent sending tasks. 0 means unlimited.
if err != nil {
  // failed to send the chord
  // do something with the error
}
```

上面的示例并行执行 task1 和 task2，聚合它们的结果并将它们传递给 task3。因此最终会发生的是:

```
multiply(add(1, 1), add(5, 5))
```

更明确:

```
(1 + 1) * (5 + 5) = 2 * 10 = 20
```

`SendChord` 返回 `ChordAsyncResult`，它跟随 AsyncResult 的接口。所以你可以做一个阻塞调用并等待回调的结果:



```go
results, err := chordAsyncResult.Get(time.Duration(time.Millisecond * 5))
if err != nil {
  // getting result of a chord failed
  // do something with the error
}
for _, result := range results {
  fmt.Println(result.Interface())
}
```

#### Chains


Chain` 只是一组任务的集合，每个成功的任务都会触发链中的下一个任务。例如:

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

signature3 := tasks.Signature{
  Name: "multiply",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 4,
    },
  },
}

chain, _ := tasks.NewChain(&signature1, &signature2, &signature3)
chainAsyncResult, err := server.SendChain(chain)
if err != nil {
  // failed to send the chain
  // do something with the error
}
```

上面的例子执行 task1，然后是 task2，然后是 task3。当一个任务成功完成时，结果被添加到链中下一个任务的参数列表的末尾。因此最终会发生的是:

```
multiply(4, add(5, 5, add(1, 1)))
```

More explicitly:

```
  4 * (5 + 5 + (1 + 1))   # task1: add(1, 1)        returns 2
= 4 * (5 + 5 + 2)         # task2: add(5, 5, 2)     returns 12
= 4 * (12)                # task3: multiply(4, 12)  returns 48
= 48
```

`SendChain` 返回 `ChainAsyncResult` ，它跟随 AsyncResult 的接口。所以你可以做一个阻塞调用，并等待整个链的结果:

```go
results, err := chainAsyncResult.Get(time.Duration(time.Millisecond * 5))
if err != nil {
  // getting result of a chain failed
  // do something with the error
}
for _, result := range results {
  fmt.Println(result.Interface())
}
```

### Periodic Tasks & Workflows

Machinery 现在支持定期任务和工作流的调度。

#### Periodic Tasks

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
)

signature := &tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}
err := server.RegisterPeriodicTask("0 6 * * ?", "periodic-task", signature)
if err != nil {
  // failed to register periodic task
}
```

#### Periodic Groups

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

group, _ := tasks.NewGroup(&signature1, &signature2)
err := server.RegisterPeriodicGroup("0 6 * * ?", "periodic-group", group)
if err != nil {
  // failed to register periodic group
}
```

#### Periodic Chains

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

signature3 := tasks.Signature{
  Name: "multiply",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 4,
    },
  },
}

chain, _ := tasks.NewChain(&signature1, &signature2, &signature3)
err := server.RegisterPeriodicChain("0 6 * * ?", "periodic-chain", chain)
if err != nil {
  // failed to register periodic chain
}
```

#### Chord

```go
import (
  "github.com/RichardKnop/machinery/v1/tasks"
  "github.com/RichardKnop/machinery/v1"
)

signature1 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 1,
    },
    {
      Type:  "int64",
      Value: 1,
    },
  },
}

signature2 := tasks.Signature{
  Name: "add",
  Args: []tasks.Arg{
    {
      Type:  "int64",
      Value: 5,
    },
    {
      Type:  "int64",
      Value: 5,
    },
  },
}

signature3 := tasks.Signature{
  Name: "multiply",
}

group := tasks.NewGroup(&signature1, &signature2)
chord, _ := tasks.NewChord(group, &signature3)
err := server.RegisterPeriodicChord("0 6 * * ?", "periodic-chord", chord)
if err != nil {
  // failed to register periodic chord
}
```

### Development

#### Requirements

* Go
* RabbitMQ (optional)
* Redis
* Memcached (optional)
* MongoDB (optional)

On OS X systems, you can install requirements using [Homebrew](http://brew.sh/):

```sh
brew install go
brew install rabbitmq
brew install redis
brew install memcached
brew install mongodb
```

Or optionally use the corresponding [Docker](http://docker.io/) containers:

```
docker run -d -p 5672:5672 rabbitmq
docker run -d -p 6379:6379 redis
docker run -d -p 11211:11211 memcached
docker run -d -p 27017:27017 mongo
docker run -d -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one:latest
```

#### Dependencies

Since Go 1.11, a new recommended dependency management system is via [modules](https://github.com/golang/go/wiki/Modules).

This is one of slight weaknesses of Go as dependency management is not a solved problem. 
Previously Go was officially recommending to use the [dep tool](https://github.com/golang/dep) but that has been abandoned now in favor of modules.

#### Testing

Easiest (and platform agnostic) way to run tests is via `docker-compose`:

```sh
make ci
```

This will basically run docker-compose command:

```sh
(docker-compose -f docker-compose.test.yml -p machinery_ci up --build -d) && (docker logs -f machinery_sut &) && (docker wait machinery_sut)
```

Alternative approach is to setup a development environment on your machine.

In order to enable integration tests, you will need to install all required services (RabbitMQ, Redis, Memcache, MongoDB) and export these environment variables:

```sh
export AMQP_URL=amqp://guest:guest@localhost:5672/
export REDIS_URL=localhost:6379
export MEMCACHE_URL=localhost:11211
export MONGODB_URL=localhost:27017
```

To run integration tests against an SQS instance, you will need to create a "test_queue" in SQS and export these environment variables:

```sh
export SQS_URL=https://YOUR_SQS_URL
export AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY
export AWS_DEFAULT_REGION=YOUR_AWS_DEFAULT_REGION
```

Then just run:

```sh
make test
```

If the environment variables are not exported, `make test` will only run unit tests.
