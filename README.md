# VEMU Orchestrator & UI

## 业务定位 (Business Positioning)

本服务是 VEMU 系统的 **"大脑"**，扮演 **"业务流程总指挥"** 和 **"用户交互界面"** 的双重角色。

## 核心职责 (Core Responsibility)

1.  **流程编排 (Orchestration)**: 作为所有其他后台服务的唯一客户端，负责将用户的单个高级请求（例如"运行5G的PBCH解码应用"）转化为对其他一系列服务的有序调用。
2.  **用户界面 (User Interface)**: 提供一个 Web 界面，允许用户提交任务、监控执行状态并查看最终结果。

### 业务边缘 (Business Boundary)

- **输入 (Input)**: 来自用户通过 UI 提交的高级任务请求。
- **输出 (Output)**:
    - 对其他服务的 API 调用 (e.g., gRPC, REST)。
    - 最终呈现给用户的仿真结果或状态。

## 流程示例 (Example Workflow)

1.  用户通过 UI 提交一个应用执行请求。
2.  Orchestrator 接收请求，并向 `vemu-dsl-service` 发送应用描述文件。
3.  `vemu-dsl-service` 返回解析后的任务依赖图 (DAG)。
4.  Orchestrator 将 DAG 和硬件资源信息发送给 `vemu-scheduling-service`。
5.  `vemu-scheduling-service` 返回一个详细的执行时间表。
6.  Orchestrator 根据时间表，向 `vemu-simulation-service` 发送一系列原子的 gRPC 指令来加载程序、执行任务、读写内存等。
7.  Orchestrator 收集所有结果，并在 UI 上呈现给用户。

## 分析 (Analysis)

这个服务的存在，完美地避免了让各个底层服务之间产生混乱的、网状的调用关系，从而防止了系统演变成难以维护的"分布式单体"。它管理着整个业务流程的状态，是实现业务价值的核心。 