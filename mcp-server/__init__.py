#!/usr/bin/env python3
"""
睿乐大脑 MCP Server Package

A Model Context Protocol server that provides access to the 睿乐大脑 knowledge management API.
"""

__version__ = "1.0.0"
__author__ = "睿乐大脑 Team"
__description__ = "睿乐大脑 MCP Server - Model Context Protocol server for 睿乐大脑 API"

from .weknora_mcp_server import WeKnoraClient, run

__all__ = ["WeKnoraClient", "run"]
