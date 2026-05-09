import sys
import os

# Add src to path
sys.path.append(os.path.join(os.path.dirname(__file__), 'nexusstate/src'))

from hippocampus.central_executive import mcp
import json

tools = mcp._mcp_server.list_tools()
print(json.dumps([{"name": t.name, "inputSchema": t.inputSchema} for t in tools], indent=2))
