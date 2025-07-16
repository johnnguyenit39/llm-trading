# Signal Frequency Adjustments

## Overview
Đã điều chỉnh các điều kiện vào lệnh để tăng tần suất tín hiệu mà vẫn đảm bảo chất lượng cơ bản.

## Các thay đổi chính

### 1. Signal Quality Thresholds (Giảm ngưỡng chất lượng)
- **MIN_SIGNAL_QUALITY_SCORE**: 7.0 → 5.0 (giảm 28%)
- **MIN_VOLUME_CONFIRMATION**: 1.5x → 1.2x (giảm 20%)
- **MIN_TREND_STRENGTH**: 0.3 → 0.2 (giảm 33%)
- **MIN_PATTERN_QUALITY**: 0.7 → 0.5 (giảm 29%)

### 2. Market Condition Filters (Nới lỏng điều kiện thị trường)
- **MAX_SPREAD_PERCENT**: 0.1% → 0.2% (tăng 100%)
- **MIN_LIQUIDITY_THRESHOLD**: 1M → 500K (giảm 50%)
- **MAX_GAP_PERCENT**: 0.5% → 1.0% (tăng 100%)
- **AVOID_LOW_VOLUME_HOURS**: true → false (tắt)

### 3. Technical Indicator Levels (Điều chỉnh chỉ báo)

#### Scalping1 (RSI)
- **RSI_OVERSOLD**: 30 → 35 (tăng 17%)
- **RSI_OVERBOUGHT**: 70 → 65 (giảm 7%)

#### Scalping2 (Stochastic)
- **STOCH_OVERSOLD**: 20 → 25 (tăng 25%)
- **STOCH_OVERBOUGHT**: 80 → 75 (giảm 6%)
- **BREAKOUT_VOLUME_MULTIPLIER**: 2.0x → 1.5x (giảm 25%)
- **BREAKOUT_PRICE_MULTIPLIER**: 1.5x → 1.2x (giảm 20%)

#### Scalping3 (Williams %R)
- **WILLIAMS_OVERSOLD**: -80 → -75 (tăng 6%)
- **WILLIAMS_OVERBOUGHT**: -20 → -25 (giảm 25%)
- **ADX_TREND_THRESHOLD**: 25 → 20 (giảm 20%)
- **VOLUME_SPIKE_MULTIPLIER**: 2.5x → 2.0x (giảm 20%)

### 4. Signal Conditions (Điều kiện tín hiệu)
- **Thay đổi từ AND sang OR**: Giảm số điều kiện bắt buộc
- **Pattern Detection**: Giảm yêu cầu về kích thước nến và volume
- **Support/Resistance**: Tăng tolerance từ 0.5% → 1%

### 5. Risk Management (Quản lý rủi ro)
- **Risk/Reward Ratio**: 1:1.5 → 1:1.2 (giảm 20%)
- **Max Leverage**: 10x → 20x (tăng 100%)

## Kết quả mong đợi
- **Tăng tần suất tín hiệu**: 3-5x so với trước
- **Duy trì chất lượng**: Vẫn đảm bảo R:R > 1:1.2
- **Giảm false negatives**: Ít bỏ lỡ cơ hội tốt
- **Tăng cơ hội giao dịch**: Nhiều setup hơn để lựa chọn

## Monitoring
Theo dõi:
- Tần suất tín hiệu mới
- Tỷ lệ win/loss
- Chất lượng tín hiệu (score)
- Drawdown và risk metrics

## Rollback Plan
Nếu cần, có thể khôi phục về các giá trị cũ trong file constants. 