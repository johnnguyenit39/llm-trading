from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import Optional, List, Dict, Any
import uvicorn
import platform
import os
import asyncio
import json
from datetime import datetime

# Environment variable loading
from dotenv import load_dotenv
load_dotenv()

# Try to import the binary options tools library
try:
    import binaryoptionstoolsv2 as bot  # type: ignore
    from BinaryOptionsToolsV2.pocketoption import PocketOption, PocketOptionAsync  # type: ignore
    from BinaryOptionsToolsV2.tracing import start_logs  # type: ignore
    BINARY_TOOLS_AVAILABLE = True
    BINARY_TOOLS_ERROR = None
except ImportError as e:
    try:
        import BinaryOptionsToolsV2 as bot  # type: ignore
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

# Get SSID from environment
SSID_KEY = os.getenv("SSID_KEY")

# Global client instance
pocket_client: Optional["PocketOptionAsync"] = None

# Pydantic models for request/response
class TradeRequest(BaseModel):
    asset: str
    amount: float
    direction: str  # "up" or "down"
    duration: Optional[int] = 60  # seconds

class TradeResponse(BaseModel):
    success: bool
    trade_id: Optional[str] = None
    message: str
    details: Optional[Dict[str, Any]] = None

class CandlesRequest(BaseModel):
    asset: str
    timeframe: Optional[int] = 60  # seconds
    count: Optional[int] = 100

class HistoryRequest(BaseModel):
    asset: str
    count: Optional[int] = 50

app = FastAPI(
    title="PocketOption Trading API",
    description="FastAPI backend with complete PocketOptionAsync integration",
    version="2.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

async def get_client() -> "PocketOptionAsync":
    """Get or create PocketOption client instance"""
    global pocket_client
    
    if not BINARY_TOOLS_AVAILABLE:
        raise HTTPException(status_code=503, detail="BinaryOptionsToolsV2 library not available")
    
    if not SSID_KEY:
        raise HTTPException(status_code=400, detail="SSID_KEY not found in environment variables")
    
    if pocket_client is None:
        pocket_client = PocketOptionAsync(ssid=SSID_KEY)
        # Give it time to initialize
        await asyncio.sleep(2)
    
    return pocket_client

# ============================================================================
# BASIC ENDPOINTS
# ============================================================================

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "message": "PocketOption Trading API is running",
        "library_available": BINARY_TOOLS_AVAILABLE,
        "ssid_configured": bool(SSID_KEY),
        "platform": platform.system()
    }

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "library_available": BINARY_TOOLS_AVAILABLE,
        "ssid_configured": bool(SSID_KEY),
        "platform_compatible": platform.system() == "Windows"
    }

@app.get("/status")
async def get_status():
    """Get detailed status information"""
    return {
        "library": {
            "available": BINARY_TOOLS_AVAILABLE,
            "error": BINARY_TOOLS_ERROR,
            "platform_support": "Windows only",
            "current_platform": platform.system()
        },
        "configuration": {
            "ssid_configured": bool(SSID_KEY),
            "ssid_preview": f"{SSID_KEY[:10]}..." if SSID_KEY else None
        },
        "client": {
            "initialized": pocket_client is not None,
            "ready": BINARY_TOOLS_AVAILABLE and bool(SSID_KEY)
        }
    }

# ============================================================================
# TRADE OPERATIONS
# ============================================================================

@app.post("/trading/buy", response_model=TradeResponse)
async def place_buy_trade(trade_request: TradeRequest):
    """Place a buy trade asynchronously"""
    try:
        client = await get_client()
        
        result = await client.buy(
            asset=trade_request.asset,
            amount=trade_request.amount,
            direction=trade_request.direction,
            duration=trade_request.duration
        )
        
        return TradeResponse(
            success=True,
            trade_id=str(result.get("id")) if isinstance(result, dict) else None,
            message="Buy trade placed successfully",
            details=result if isinstance(result, dict) else {"result": str(result)}
        )
        
    except Exception as e:
        return TradeResponse(
            success=False,
            message=f"Failed to place buy trade: {str(e)}"
        )

@app.post("/trading/sell", response_model=TradeResponse)
async def place_sell_trade(trade_request: TradeRequest):
    """Place a sell trade asynchronously"""
    try:
        client = await get_client()
        
        result = await client.sell(
            asset=trade_request.asset,
            amount=trade_request.amount,
            direction=trade_request.direction,
            duration=trade_request.duration
        )
        
        return TradeResponse(
            success=True,
            trade_id=str(result.get("id")) if isinstance(result, dict) else None,
            message="Sell trade placed successfully",
            details=result if isinstance(result, dict) else {"result": str(result)}
        )
        
    except Exception as e:
        return TradeResponse(
            success=False,
            message=f"Failed to place sell trade: {str(e)}"
        )

@app.get("/trading/check-win/{trade_id}")
async def check_trade_outcome(trade_id: str):
    """Check the outcome of a trade ('win', 'draw', or 'loss')"""
    try:
        client = await get_client()
        result = await client.check_win(trade_id)
        
        return {
            "trade_id": trade_id,
            "outcome": result,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to check trade outcome: {str(e)}")

# ============================================================================
# MARKET DATA
# ============================================================================

@app.post("/market/candles")
async def get_candles(request: CandlesRequest):
    """Fetch historical candle data"""
    try:
        client = await get_client()
        
        candles = await client.get_candles(
            asset=request.asset,
            timeframe=request.timeframe,
            count=request.count
        )
        
        return {
            "asset": request.asset,
            "timeframe": request.timeframe,
            "count": len(candles) if candles else 0,
            "candles": candles,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch candles: {str(e)}")

@app.post("/market/history")
async def get_asset_history(request: HistoryRequest):
    """Retrieve recent data for a specific asset"""
    try:
        client = await get_client()
        
        history = await client.history(
            asset=request.asset,
            count=request.count
        )
        
        return {
            "asset": request.asset,
            "count": len(history) if history else 0,
            "history": history,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch history: {str(e)}")

# ============================================================================
# ACCOUNT MANAGEMENT
# ============================================================================

@app.get("/account/balance")
async def get_account_balance():
    """Get current account balance"""
    try:
        client = await get_client()
        balance = await client.balance()
        
        return {
            "balance": balance,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch balance: {str(e)}")

@app.get("/account/opened-deals")
async def get_opened_deals():
    """Get all open trades"""
    try:
        client = await get_client()
        deals = await client.opened_deals()
        
        return {
            "count": len(deals) if deals else 0,
            "deals": deals,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch opened deals: {str(e)}")

@app.get("/account/closed-deals")
async def get_closed_deals():
    """Get all closed trades"""
    try:
        client = await get_client()
        deals = await client.closed_deals()
        
        return {
            "count": len(deals) if deals else 0,
            "deals": deals,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch closed deals: {str(e)}")

@app.get("/account/payout")
async def get_payout_percentages():
    """Get payout percentages"""
    try:
        client = await get_client()
        payout = await client.payout()
        
        return {
            "payout": payout,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to fetch payout: {str(e)}")

# ============================================================================
# REAL-TIME DATA
# ============================================================================

@app.get("/realtime/subscribe/{asset}")
async def subscribe_to_symbol(asset: str):
    """Subscribe to real-time candle updates for an asset"""
    async def generate_data():
        try:
            client = await get_client()
            
            async for candle in client.subscribe_symbol(asset):
                data = {
                    "asset": asset,
                    "candle": candle,
                    "timestamp": datetime.now().isoformat()
                }
                yield f"data: {json.dumps(data)}\n\n"
                
        except Exception as e:
            error_data = {
                "error": str(e),
                "timestamp": datetime.now().isoformat()
            }
            yield f"data: {json.dumps(error_data)}\n\n"
    
    return StreamingResponse(
        generate_data(),
        media_type="text/plain",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"}
    )

@app.get("/realtime/subscribe-timed/{asset}")
async def subscribe_to_symbol_timed(asset: str, interval: int = 60):
    """Subscribe to timed real-time candle updates"""
    async def generate_data():
        try:
            client = await get_client()
            
            async for candle in client.subscribe_symbol_timed(asset, interval):
                data = {
                    "asset": asset,
                    "interval": interval,
                    "candle": candle,
                    "timestamp": datetime.now().isoformat()
                }
                yield f"data: {json.dumps(data)}\n\n"
                
        except Exception as e:
            error_data = {
                "error": str(e),
                "timestamp": datetime.now().isoformat()
            }
            yield f"data: {json.dumps(error_data)}\n\n"
    
    return StreamingResponse(
        generate_data(),
        media_type="text/plain",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"}
    )

@app.get("/realtime/subscribe-chunked/{asset}")
async def subscribe_to_symbol_chunked(asset: str, chunk_size: int = 10):
    """Subscribe to chunked real-time candle updates"""
    async def generate_data():
        try:
            client = await get_client()
            
            async for chunk in client.subscribe_symbol_chunked(asset, chunk_size):
                data = {
                    "asset": asset,
                    "chunk_size": chunk_size,
                    "chunk": chunk,
                    "timestamp": datetime.now().isoformat()
                }
                yield f"data: {json.dumps(data)}\n\n"
                
        except Exception as e:
            error_data = {
                "error": str(e),
                "timestamp": datetime.now().isoformat()
            }
            yield f"data: {json.dumps(error_data)}\n\n"
    
    return StreamingResponse(
        generate_data(),
        media_type="text/plain",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"}
    )

# ============================================================================
# UTILITY ENDPOINTS
# ============================================================================

@app.post("/utils/initialize-logging")
async def initialize_logging(log_path: str = "logs/", log_level: str = "INFO", terminal: bool = True):
    """Initialize application logging"""
    try:
        if BINARY_TOOLS_AVAILABLE:
            start_logs(path=log_path, level=log_level, terminal=terminal)
            return {
                "success": True,
                "message": "Logging initialized successfully",
                "config": {
                    "path": log_path,
                    "level": log_level,
                    "terminal": terminal
                }
            }
        else:
            raise HTTPException(status_code=503, detail="Library not available")
            
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to initialize logging: {str(e)}")

@app.get("/docs-info")
async def get_documentation():
    """Get comprehensive API documentation"""
    return {
        "endpoints": {
            "trading": {
                "POST /trading/buy": "Place buy trade",
                "POST /trading/sell": "Place sell trade", 
                "GET /trading/check-win/{trade_id}": "Check trade outcome"
            },
            "market_data": {
                "POST /market/candles": "Get historical candles",
                "POST /market/history": "Get asset history"
            },
            "account": {
                "GET /account/balance": "Get account balance",
                "GET /account/opened-deals": "Get open trades",
                "GET /account/closed-deals": "Get closed trades",
                "GET /account/payout": "Get payout percentages"
            },
            "realtime": {
                "GET /realtime/subscribe/{asset}": "Real-time candle updates",
                "GET /realtime/subscribe-timed/{asset}": "Timed real-time updates",
                "GET /realtime/subscribe-chunked/{asset}": "Chunked real-time updates"
            }
        },
        "setup": {
            "environment": "Create .env file with SSID_KEY=your_session_id",
            "platform": "Run on Windows machine",
            "library": "pip install binaryoptionstoolsv2==0.1.7"
        }
    }

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000) 
    