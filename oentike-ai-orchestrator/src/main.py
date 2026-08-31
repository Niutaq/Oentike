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
    events_buffer: List[dict]
    analysis_result: dict

ollama_model = os.getenv("OLLAMA_MODEL", "qwen3-vl:4b")
print(f"Initializing ChatOllama with model: {ollama_model}")
llm = ChatOllama(model=ollama_model, temperature=0.1, format="json")

class ThreatAnalysis(BaseModel):
    is_malicious: bool = Field(description="True if the batch of events contains malicious behavior or confirmed threats.")
    reasoning: str = Field(description="Detailed explanation of the findings, threat actor TTPs (Tactics, Techniques, and Procedures), and recommended SecOps mitigations.")
    risk_level: str = Field(description="CRITICAL, HIGH, MEDIUM, or LOW.")
    recommended_action: str = Field(description="QUARANTINE, MONITOR, or IGNORE.")

async def analyzer_node(state: GraphState) -> GraphState:
    events = state["events_buffer"]
    
    parser = PydanticOutputParser(pydantic_object=ThreatAnalysis)
    
    prompt = f"""
You are an expert SOC Analyst and Cyber Threat Intelligence (CTI) expert.
Analyze the following recent security events (telemetry, WAF logs, mTLS connections) from our Zero-Trust Control Plane.
Look for anomalous patterns, malicious User-Agents, SQLi/XSS attempts, or high-risk behavior.
Provide a detailed threat intelligence report.

Events:
{json.dumps(events, indent=2)}

{parser.get_format_instructions()}
"""
    try:
        print(f"[SOC Analyzer] Analyzing {len(events)} events with {ollama_model}...")
        result = await llm.ainvoke([
            SystemMessage(content="You are a SOC AI analyzer. Output valid JSON only."),
            HumanMessage(content=prompt)
        ])
        parsed = parser.invoke(result)
        analysis = parsed.dict()
    except Exception as e:
        print(f"[SOC Analyzer Error] {e}")
        analysis = {
            "is_malicious": False,
            "reasoning": "Failed to analyze events due to an error.",
            "risk_level": "LOW",
            "recommended_action": "MONITOR"
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
    
    stream_name = "SECOPS_EVENTS"
    try:
        await js.add_stream(name=stream_name, subjects=["SECOPS.>"])
    except Exception:
        pass

    app = build_graph()
    
    events_buffer = []
    last_analysis_time = time.time()
        
    async def message_handler(msg):
        nonlocal events_buffer, last_analysis_time
        try:
            req_data = json.loads(msg.data.decode())
        except Exception as e:
            print("Failed to decode message:", e)
            await msg.nak()
            return

        # Fast path: Forward raw telemetry immediately to SECOPS.ai_processed so Astro UI updates in real-time
        risk_score = req_data.get("risk_score", 0)
        status = "QUARANTINED" if risk_score > 80 else "SECURED"
        
        extracted_data = {
            "agent_id": req_data.get("agent_id", "Unknown"),
            "status": status,
            "routed_provider": "zero_trust_edge",
            "risk_score": risk_score,
            "threat_category": "Anomaly",
            "metrics": {
                "client_mtls_setup_ms": req_data.get("client_mtls_setup_ms", 0),
                "payload_bytes_out": req_data.get("payload_bytes_out", 0),
                "server_proc_time_us": req_data.get("server_proc_time_us", 0)
            },
            "workload": {
                "type": req_data.get("workload_type", "Telemetry Stream"),
                "cpu_cycles": req_data.get("estimated_cpu_cycles", 0),
                "memory_mb": req_data.get("estimated_memory_mb", 0)
            },
            "reasoning": "Real-time edge telemetry processed."
        }
        
        await js.publish("SECOPS.ai_processed", json.dumps(extracted_data).encode())
        await msg.ack()
        
        # Add to buffer for AI periodic analysis
        events_buffer.append(req_data)
        
        # Periodically invoke the LLM
        if len(events_buffer) >= 10 or (time.time() - last_analysis_time > 20 and len(events_buffer) > 0):
            buffer_copy = events_buffer[:]
            events_buffer = []
            last_analysis_time = time.time()
            
            # Fire and forget LLM task so it doesn't block the stream
            asyncio.create_task(run_analysis_and_publish(app, js, buffer_copy))

    print("Subscribing to SECOPS.metrics...")
    sub = await js.subscribe("SECOPS.metrics", cb=message_handler)
    print("SecOps SOC Orchestrator is running...")
    
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        await sub.unsubscribe()
        await nc.close()

async def run_analysis_and_publish(app, js, events):
    try:
        result_state = await app.ainvoke({"events_buffer": events})
        analysis = result_state.get("analysis_result", {})
        
        insight_data = {
            "agent_id": "SOC-AI-Orchestrator",
            "status": "THREAT_DETECTED" if analysis.get("is_malicious") else "CLEAN",
            "routed_provider": "Threat Intelligence",
            "risk_score": 100 if analysis.get("risk_level") == "CRITICAL" else (80 if analysis.get("risk_level") == "HIGH" else 0),
            "threat_category": f"Risk: {analysis.get('risk_level')}",
            "metrics": {
                "client_mtls_setup_ms": 0,
                "payload_bytes_out": 0,
                "server_proc_time_us": 0
            },
            "workload": {
                "type": f"Action: {analysis.get('recommended_action')}",
                "cpu_cycles": 0,
                "memory_mb": 0
            },
            "reasoning": analysis.get("reasoning", "No insight generated.")
        }
        await js.publish("SECOPS.ai_processed", json.dumps(insight_data).encode())
        print("Published SOC Insight to SECOPS.ai_processed")
    except Exception as e:
        print(f"Error running analysis task: {e}")

if __name__ == '__main__':
    asyncio.run(main())
