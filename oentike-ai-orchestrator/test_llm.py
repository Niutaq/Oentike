import asyncio
from langchain_core.messages import HumanMessage
from langchain_ollama import ChatOllama
from pydantic import BaseModel, Field

class AnalysisOutput(BaseModel):
    expense_title: str = Field(description="A short title for the extracted expense")
    amount: float = Field(description="The exact or estimated cost of the expense")
    currency: str = Field(description="The currency (e.g., USD)")
    category: str = Field(description="FOCUS category (e.g., Compute, Storage, Software)")
    business_value: str = Field(description="Evaluation of the business value and ROI")
    risk_level: str = Field(description="Risk level of this expense (Low, Medium, High)")
    tech_cost_assessment: str = Field(description="Technical evaluation of request's network and CPU overhead")

llm = ChatOllama(model="qwen3.5:1.4b", temperature=0.1)

prompt = """
You are an expert FinOps Analyst AI adhering to the FOCUS (FinOps Open Cost & Usage Specification) standard. 
Your task is to process user chat requests regarding infrastructure or software expenses.
Extract the expense details, evaluate the business justification, assign a FOCUS category, and determine the risk level.

Additionally, evaluate the technical overhead of this request itself based on the following telemetry:
- mTLS Setup Time: 10 ms
- Payload Size: 100 bytes
- Server Processing Time: 5 ms

User Chat Message:
"I need a new AWS EC2 instance for my project, it will cost around 50 bucks."

If the user didn't specify the exact amount, provide a realistic estimated amount and currency.
Provide a short `tech_cost_assessment` analyzing if the telemetry overhead is acceptable.
"""

structured_llm = llm.with_structured_output(AnalysisOutput)
try:
    result = structured_llm.invoke([HumanMessage(content=prompt)])
    print("SUCCESS")
    print(result.dict())
except Exception as e:
    print("FAILED")
    print(e)
