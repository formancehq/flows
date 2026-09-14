# Stage

One step of a workflow, whose shape depends on the stage type


## Supported Types

### StageSend

```go
stage := components.CreateStageStageSend(components.StageSend{/* values here */})
```

### StageDelay

```go
stage := components.CreateStageStageDelay(components.StageDelay{/* values here */})
```

### StageWaitEvent

```go
stage := components.CreateStageStageWaitEvent(components.StageWaitEvent{/* values here */})
```

### Update

```go
stage := components.CreateStageUpdate(components.Update{/* values here */})
```

