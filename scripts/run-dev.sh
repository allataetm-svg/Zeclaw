#!/bin/bash
set -e
echo "Starting Zeclaw backend in development mode..."
cd backend
go run ./cmd/zeclaw/
