import { useState } from 'react';
import './App.css';

function App() {
  // 定义组件的状态
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);

  // 点击按钮时触发的函数
  const handleRunSimulation = async () => {
    // 重置状态
    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      // 向后端API发送POST请求
      const response = await fetch('/api/v1/simulations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        // 发送一个模拟的应用包ID
        body: JSON.stringify({ appPackageId: '5g-pdsch-mock' }),
      });

      // 检查响应是否成功
      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Backend returned status ${response.status}: ${errorText}`);
      }

      // 解析JSON响应并更新状态
      const data = await response.json();
      setResult(data);

    } catch (err) {
      // 捕获网络错误或解析错误
      setError(err.message);
    } finally {
      // 无论成功或失败，都结束加载状态
      setIsLoading(false);
    }
  };

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>VEMU Orchestrator UI</h1>
        <p>A unified interface to orchestrate the VEMU simulation workflow.</p>
      </header>
      
      <main className="app-main">
        <button 
          className="run-button"
          onClick={handleRunSimulation} 
          disabled={isLoading}
        >
          {isLoading ? 'Orchestrating...' : 'Run Full Simulation Workflow'}
        </button>

        <div className="result-container">
          {/* 根据不同状态显示不同内容 */}
          {isLoading && <p className="loading-text">Loading... Please wait.</p>}
          {error && <div className="error-box">
            <h3>An Error Occurred</h3>
            <pre>{error}</pre>
          </div>}
          {result && <div className="result-box">
            <h3>Orchestration Successful</h3>
            <p>{result.message}</p>
            <pre>{JSON.stringify(result, null, 2)}</pre>  
          </div>}
        </div>
      </main>
    </div>
  );
}

export default App; 