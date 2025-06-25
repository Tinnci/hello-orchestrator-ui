package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	// 引入 gRPC 和生成的 proto 包
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "vemu-orchestrator.com/backend/proto"
)

// --- 常量定义 ---
const (
	DSLServiceURL        = "http://localhost:8001/v1/parse"
	SchedulingServiceURL = "http://localhost:8002/v1/schedule"
	SimulationServiceURL = "localhost:50051" // gRPC URL
	DefaultTimeout       = 10 * time.Second
)

// --- 数据结构定义 (与之前类似，但会添加最终的响应结构) ---

// SimulationRequest: 从前端接收的启动仿真请求
type SimulationRequest struct {
	AppPackageID string `json:"appPackageId"`
}

// DSLServiceRequest: 发送给 DSL 服务的请求
type DSLServiceRequest struct {
	DSLText string `json:"dsl_text"`
}

// DAGNode, DAGEdge, DAG: 用于解码 DSL 服务返回的 DAG 结构
type DAGNode struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceFile string `json:"source_file"`
}
type DAGEdge struct {
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	DataSize int    `json:"data_size"`
}
type DAG struct {
	Nodes []DAGNode `json:"nodes"`
	Edges []DAGEdge `json:"edges"`
}
type DSLServiceResponse struct {
	Dag DAG `json:"dag"`
}

// Core, Resources: 用于构建发送给调度服务请求中的资源部分
type Core struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}
type Resources struct {
	Cores        []Core `json:"cores"`
	MemorySizeKb int    `json:"memorySizeKb"`
}

// SchedulingServiceRequest: 发送给调度服务的请求
type SchedulingServiceRequest struct {
	Dag       DAG       `json:"dag"`
	Resources Resources `json:"resources"`
}

// ScheduledTask, ScheduleResponse: 用于解码调度服务返回的最终调度计划
type ScheduledTask struct {
	TaskID     string `json:"taskId"`
	CoreID     int    `json:"coreId"`
	StartCycle int    `json:"startCycle"`
	Inputs     []any  `json:"inputs"`  // 使用 any 简化，因为暂时不关心其内部结构
	Outputs    []any  `json:"outputs"` // 使用 any 简化
}
type SchedulingServiceResponse struct {
	Schedule []ScheduledTask `json:"schedule"`
}

// FinalResponse: 编排器返回给前端的最终响应
type FinalResponse struct {
	Schedule         *SchedulingServiceResponse `json:"schedule"`
	SimulationStatus string                     `json:"simulationStatus"`
	InitialCycle     uint64                     `json:"initialCycle"`
	Message          string                     `json:"message"`
}

// --- HTTP 处理器 ---

// handleSimulations 是核心编排逻辑
func handleSimulations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}
	log.Printf("Orchestration started for App Package: %s", req.AppPackageID)

	// 创建一个带超时的HTTP客户端
	client := &http.Client{Timeout: DefaultTimeout}

	// --- 步骤 1: 调用 DSL 服务 ---
	// TODO: 从 AppPackageID 获取真实的 DSL 文本，目前使用硬编码的占位符
	mockDSLText := `dag HelloWorld = { [out_var] = TaskA(), [final_result] = TaskB(out_var) }`
	dag, err := getDAGFromDSLService(client, mockDSLText)
	if err != nil {
		log.Printf("Error from DSL Service: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get DAG from DSL service: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully got DAG with %d nodes and %d edges.", len(dag.Nodes), len(dag.Edges))

	// --- 步骤 2: 调用调度服务 ---
	// TODO: 获取真实的资源信息，目前使用硬编码的占位符
	mockResources := Resources{
		Cores:        []Core{{ID: 0, Type: "riscv_core"}},
		MemorySizeKb: 8192,
	}
	schedule, err := getScheduleFromSchedulingService(client, *dag, mockResources)
	if err != nil {
		log.Printf("Error from Scheduling Service: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get schedule: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully got schedule with %d tasks.", len(schedule.Schedule))

	// --- 步骤 3: 调用仿真服务 (gRPC) ---
	// TODO: 实现 gRPC 客户端逻辑，根据 schedule 调用 simulation-service
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	// 建立到 gRPC 服务器的连接
	conn, err := grpc.DialContext(ctx, SimulationServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Printf("Failed to connect to simulation service: %v", err)
		http.Error(w, fmt.Sprintf("gRPC connection error: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	log.Printf("Successfully connected to gRPC simulation service at %s", SimulationServiceURL)

	// 创建 gRPC 客户端
	simClient := pb.NewVemuServiceClient(conn)

	// 调用 Reset RPC
	resetStatus, err := simClient.Reset(ctx, &pb.Empty{})
	if err != nil || !resetStatus.Ok {
		log.Printf("Failed to reset simulator: %v (status: %v)", err, resetStatus)
		http.Error(w, "Failed to reset simulator", http.StatusInternalServerError)
		return
	}
	log.Println("Simulator has been reset.")

	// 调用 QueryState RPC
	initialState, err := simClient.QueryState(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("Failed to query simulator state: %v", err)
		http.Error(w, "Failed to query simulator state", http.StatusInternalServerError)
		return
	}
	log.Printf("Initial state queried. Cycle count: %d", initialState.GetCycle())

	// TODO: 根据调度计划，实现更复杂的调用流程 (LoadFirmware, Run, etc.)

	// --- 最终响应 ---
	finalResp := FinalResponse{
		Schedule:         schedule,
		SimulationStatus: "Completed (mock)",
		InitialCycle:     initialState.GetCycle(),
		Message:          "Orchestration finished. Complex simulation flow not yet implemented.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResp)
}

func getDAGFromDSLService(client *http.Client, dslText string) (*DAG, error) {
	reqBody := DSLServiceRequest{DSLText: dslText}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DSL request: %w", err)
	}

	resp, err := client.Post(DSLServiceURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("request to DSL service failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DSL service returned non-OK status: %s", resp.Status)
	}

	var dslResponse DSLServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&dslResponse); err != nil {
		return nil, fmt.Errorf("failed to decode DSL response: %w", err)
	}

	return &dslResponse.Dag, nil
}

func getScheduleFromSchedulingService(client *http.Client, dag DAG, resources Resources) (*SchedulingServiceResponse, error) {
	reqBody := SchedulingServiceRequest{Dag: dag, Resources: resources}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scheduling request: %w", err)
	}

	resp, err := client.Post(SchedulingServiceURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("request to scheduling service failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scheduling service returned non-OK status: %s", resp.Status)
	}

	var scheduleResponse SchedulingServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResponse); err != nil {
		return nil, fmt.Errorf("failed to decode scheduling response: %w", err)
	}

	return &scheduleResponse, nil
}

// healthCheckHandler 用于简单的健康检查
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Orchestrator backend is running!")
}

func main() {
	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/simulations", handleSimulations)
	mux.HandleFunc("/health", healthCheckHandler)

	// 启动服务器
	port := "8080"
	log.Printf("VEMU Orchestrator backend starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
