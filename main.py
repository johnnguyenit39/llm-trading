from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

# Try to import the binary options tools library
try:
    import binaryoptionstoolsv2 as bot  # type: ignore
    BINARY_TOOLS_AVAILABLE = True
    BINARY_TOOLS_ERROR = None
except ImportError as e:
    try:
        import BinaryOptionsToolsV2 as bot
        BINARY_TOOLS_AVAILABLE = True
        BINARY_TOOLS_ERROR = None
    except ImportError as e2:
        bot = None
        BINARY_TOOLS_AVAILABLE = False
        BINARY_TOOLS_ERROR = f"Import error: {str(e)} and {str(e2)}"
except Exception as e:
    bot = None
    BINARY_TOOLS_AVAILABLE = False
    BINARY_TOOLS_ERROR = f"General error: {str(e)}"

app = FastAPI(
    title="Simple FastAPI Backend",
    description="A simple FastAPI backend with binaryoptionstoolsv2 integration",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/")
async def root():
    """Root endpoint"""
    return {"message": "Hello World! FastAPI Backend is running"}

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "message": "API is working properly"}

@app.get("/test-endpoint")
async def test_endpoint():
    """Test endpoint to verify the API is working"""
    return {
        "message": "Test endpoint working",
        "binary_tools_status": "available" if BINARY_TOOLS_AVAILABLE else "unavailable",
        "timestamp": "2024-01-01T00:00:00Z"
    }

@app.get("/binary-tools-info")
async def binary_tools_info():
    """Get information about the binaryoptionstoolsv2 library"""
    if not BINARY_TOOLS_AVAILABLE:
        return {
            "library": "binaryoptionstoolsv2",
            "version": "0.1.7",
            "status": "installation detected but import failed",
            "error": BINARY_TOOLS_ERROR,
            "note": "The library is installed but has import issues. This is common with alpha/beta versions."
        }
    
    try:
        # Try to get some basic info about the library
        return {
            "library": "binaryoptionstoolsv2",
            "version": "0.1.7",
            "status": "successfully imported",
            "available_attributes": [attr for attr in dir(bot) if not attr.startswith('_')]
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error accessing binary tools: {str(e)}")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000) 
    