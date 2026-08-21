import asyncio
import os
import json
import time
from typing import TypedDict, List
from nats.aio.client import Client as NATS

from langchain_core.messages import SystemMessage, HumanMessage
from langchain_core.output_parsers import PydanticOutputParser
from langchain_ollama import ChatOllama
from langgraph.graph import StateGraph, START, END
from pydantic import BaseModel, Field

class GraphState(TypedDict):
    metrics_buffer: List[dict]
    analysis_result: dict

ollama_model = os.getenv("OLLAMA_MODEL", "qwen3-vl:4b")
print(f"Initializing ChatOllama with model: {ollama_model}")
llm = ChatOllama(model=ollama_model, temperature=0.1, format="json")

class AnomalyAnalysis(BaseModel):
    has_anomaly: bool = Field(description="True if there is an anomaly or high resource usage in the metrics.")
    reasoning: str = Field(description="Detailed explanation of the findings and FinOps optimization insights.")
    risk_level: str = Field(description="HIGH, MEDIUM, or LOW risk.")

async def analyzer_node(state: GraphState) -> GraphState:
    metrics = state["metrics_buffer"]
    
    parser = PydanticOutputParser(pydantic_object=AnomalyAnalysis)
    
    prompt = f"""
You are an expert Cloud FinOps Architect. Analyze the following recent hardware telemetry metrics from our Edge Agents.
Look for high CPU usage, high memory usage, or mTLS latency spikes.
Provide FinOps optimization insights.

Metrics:
{json.dumps(metrics, indent=2)}

{parser.get_format_instructions()}
"""
    try:
        print(f"[Analyzer] Analyzing {len(metrics)} events with {ollama_model}...")
        result = await llm.ainvoke([
            SystemMessage(content="You are a FinOps AI analyzer. Output valid JSON only."), 
            HumanMessage(content=prompt)
        ])
        parsed = parser.invoke(result)
        analysis = parsed.dict()
    except Exception as e:
        print(f"[Analyzer Error] {e}")
        analysis = {
            "has_anomaly": False,
            "reasoning": "Failed to analyze metrics.",
            "risk_level": "LOW"
        }
        
    return {"analysis_result": analysis}

def build_graph():
    workflow = StateGraph(GraphState)
    workflow.add_node("analyzer", analyzer_node)
    
    workflow.add_edge(START, "analyzer")
    workflow.add_edge("analyzer", END)
    
    return workflow.compile()

async def main():
    nc = NATS()
    nats_url = os.getenv("NATS_URL", "nats://localhost:4222")
    print(f"Connecting to NATS at {nats_url}...")
    await nc.connect(nats_url)
    js = nc.jetstream()
    
    stream_name = "FINOPS_EVENTS"
    try:
        await js.add_stream(name=stream_name, subjects=["FINOPS.>"])
    except Exception:
        pass

    app = build_graph()
    
    metrics_buffer = []
    last_analysis_time = time.time()
        
    async def message_handler(msg):
        nonlocal metrics_buffer, last_analysis_time
        try:
            req_data = json.loads(msg.data.decode())
        except Exception as e:
            print("Failed to decode message:", e)
            await msg.nak()
            return

        # Fast path: Forward raw telemetry immediately to FINOPS.ai_processed so Astro UI updates in real-time
        cpu_cycles = req_data.get("estimated_cpu_cycles", 0)
        mem_mb = req_data.get("estimated_memory_mb", 0)
        status = "ROUTED" if cpu_cycles < 8000 else "REJECTED_BUDGET"
        
        extracted_data = {
            "agent_id": req_data.get("agent_id", "Unknown"),
            "status": status,
            "routed_provider": "local_edge_cluster",
            "estimated_cost": (cpu_cycles / 100) * 0.010,
            "charge_category": "Compute",
            "metrics": {
                "client_mtls_setup_ms": req_data.get("client_mtls_setup_ms", 0),
                "payload_bytes_out": req_data.get("payload_bytes_out", 0),
                "server_proc_time_us": req_data.get("server_proc_time_us", 0)
            },
            "workload": {
                "type": req_data.get("workload_type", "Sensor Stream"),
                "cpu_cycles": cpu_cycles,
                "memory_mb": mem_mb
            },
            "reasoning": "Real-time telemetry within FinOps budget."
        }
        
        await js.publish("FINOPS.ai_processed", json.dumps(extracted_data).encode())
        await msg.ack()
        
        # Add to buffer for AI periodic analysis
        metrics_buffer.append(req_data)
        
        # Periodically invoke the LLM
        if len(metrics_buffer) >= 10 or (time.time() - last_analysis_time > 20 and len(metrics_buffer) > 0):
            buffer_copy = metrics_buffer[:]
            metrics_buffer = []
            last_analysis_time = time.time()
            
            # Fire and forget LLM task so it doesn't block the stream
            asyncio.create_task(run_analysis_and_publish(app, js, buffer_copy))

    print("Subscribing to FINOPS.metrics...")
    sub = await js.subscribe("FINOPS.metrics", cb=message_handler)
    print("Workload Router Engine is running...")
    
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        await sub.unsubscribe()
        await nc.close()

async def run_analysis_and_publish(app, js, metrics):
    try:
        result_state = await app.ainvoke({"metrics_buffer": metrics})
        analysis = result_state.get("analysis_result", {})
        
        insight_data = {
            "agent_id": "AI-Orchestrator",
            "status": "ANOMALY_DETECTED" if analysis.get("has_anomaly") else "OPTIMAL",
            "routed_provider": "FinOps Insight",
            "estimated_cost": 0.0,
            "charge_category": "Deep Analysis",
            "metrics": {
                "client_mtls_setup_ms": 0,
                "payload_bytes_out": 0,
                "server_proc_time_us": 0
            },
            "workload": {
                "type": f"Risk: {analysis.get('risk_level')}",
                "cpu_cycles": 0,
                "memory_mb": 0
            },
            "reasoning": analysis.get("reasoning", "No insight generated.")
        }
        await js.publish("FINOPS.ai_processed", json.dumps(insight_data).encode())
        print("Published FinOps Insight to FINOPS.ai_processed")
    except Exception as e:
        print(f"Error running analysis task: {e}")

if __name__ == '__main__':
    asyncio.run(main())
