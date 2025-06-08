# Simple FastAPI Backend

A simple FastAPI backend project with `binaryoptionstoolsv2` integration.

## Features

- FastAPI web framework
- CORS middleware enabled
- Health check endpoint
- Binary options tools integration
- Auto-reload development server
- VS Code debugging support

## Installation

1. Clone the repository:
```bash
git clone <your-repo-url>
cd be
```

2. Create and activate a virtual environment (recommended):
```bash
# Create virtual environment
python3.12 -m venv venv

# Activate virtual environment
# On macOS/Linux:
source venv/bin/activate
# On Windows:
# venv\Scripts\activate
```

3. Install dependencies:
```bash
pip install -r requirements.txt
```

## Running the Application

**Important**: Make sure your virtual environment is activated before running the application.

### Method 1: Using uvicorn directly
```bash
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

### Method 2: Using the run script
```bash
python run.py
```

### Method 3: Running main.py directly
```bash
python main.py
```

The API will be available at:
- **Main API**: http://localhost:8000
- **Interactive API docs (Swagger)**: http://localhost:8000/docs
- **Alternative API docs (ReDoc)**: http://localhost:8000/redoc

## Debugging in VS Code

This project includes VS Code debugging configurations:

1. **Python: FastAPI** - Debug using uvicorn module (recommended)
2. **Python: FastAPI (Direct)** - Debug main.py directly  
3. **Python: FastAPI (run.py)** - Debug using run.py script

To debug:
1. Set breakpoints in your code
2. Go to the Debug panel (Ctrl+Shift+D or Cmd+Shift+D)
3. Select a configuration from the dropdown
4. Press F5 or click the green play button

## API Endpoints

- `GET /` - Root endpoint with welcome message
- `GET /health` - Health check endpoint
- `GET /binary-tools-info` - Information about the binaryoptionstoolsv2 library
- `GET /test-endpoint` - Test endpoint to verify functionality

## Dependencies

- **fastapi**: Modern, fast web framework for building APIs
- **uvicorn**: ASGI server for running FastAPI applications
- **python-multipart**: For handling form data
- **binaryoptionstoolsv2**: Binary options tools library (v0.1.7)

## Development

The server runs with auto-reload enabled in development mode, so changes to your code will automatically restart the server.

## Virtual Environment

This project uses a virtual environment to isolate dependencies. The virtual environment is created in the `venv/` directory and is used by the VS Code debugger configurations.

## Project Structure

```
be/
├── venv/             # Virtual environment (created after setup)
├── main.py           # Main FastAPI application
├── run.py            # Alternative run script
├── requirements.txt  # Python dependencies
├── .vscode/          # VS Code configuration
│   └── launch.json   # Debug configurations
├── .gitignore       # Git ignore file
└── README.md        # This file
```