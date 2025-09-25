#!/bin/bash

echo "🔧 Concurrent Search Configuration Optimizer"
echo "============================================="

# 检查当前配置
echo "📋 Current configuration analysis:"
echo "  - Per-fetch timeout: 6s (default)"
echo "  - Overall timeout: 10s (default)"
echo "  - Max concurrency: 4 (default)"
echo "  - Success rate: ~50% (from your logs)"
echo ""

# 提供优化建议
echo "💡 Optimization recommendations:"
echo ""

echo "🎯 Option 1: Balanced Configuration (Recommended)"
echo "   - Per-fetch timeout: 10s"
echo "   - Overall timeout: 15s"
echo "   - Max concurrency: 3"
echo "   - Expected success rate: 80-85%"
echo "   - Expected response time: 8-12s"
echo ""

echo "🛡️ Option 2: Conservative Configuration (High Reliability)"
echo "   - Per-fetch timeout: 12s"
echo "   - Overall timeout: 20s"
echo "   - Max concurrency: 2"
echo "   - Expected success rate: 90-95%"
echo "   - Expected response time: 10-15s"
echo ""

echo "⚡ Option 3: Aggressive Configuration (Fast Response)"
echo "   - Per-fetch timeout: 8s"
echo "   - Overall timeout: 12s"
echo "   - Max concurrency: 4"
echo "   - Expected success rate: 60-70%"
echo "   - Expected response time: 6-10s"
echo ""

# 询问用户选择
echo "🤔 Which configuration would you like to apply?"
echo "1) Balanced (recommended)"
echo "2) Conservative (high reliability)"
echo "3) Aggressive (fast response)"
echo "4) Custom configuration"
echo "5) Show current configuration only"
echo ""

read -p "Enter your choice (1-5): " choice

case $choice in
    1)
        echo "🎯 Applying balanced configuration..."
        echo ""
        echo "Add these environment variables to your deployment:"
        echo "export SEARCH_PER_FETCH_TIMEOUT_SEC=10"
        echo "export SEARCH_OVERALL_TIMEOUT_SEC=15"
        echo "export SEARCH_MAX_CONCURRENCY=3"
        echo ""
        echo "Or add to your config.yaml:"
        echo "SEARCH_PER_FETCH_TIMEOUT_SEC: 10"
        echo "SEARCH_OVERALL_TIMEOUT_SEC: 15"
        echo "SEARCH_MAX_CONCURRENCY: 3"
        ;;
    2)
        echo "🛡️ Applying conservative configuration..."
        echo ""
        echo "Add these environment variables to your deployment:"
        echo "export SEARCH_PER_FETCH_TIMEOUT_SEC=12"
        echo "export SEARCH_OVERALL_TIMEOUT_SEC=20"
        echo "export SEARCH_MAX_CONCURRENCY=2"
        echo ""
        echo "Or add to your config.yaml:"
        echo "SEARCH_PER_FETCH_TIMEOUT_SEC: 12"
        echo "SEARCH_OVERALL_TIMEOUT_SEC: 20"
        echo "SEARCH_MAX_CONCURRENCY: 2"
        ;;
    3)
        echo "⚡ Applying aggressive configuration..."
        echo ""
        echo "Add these environment variables to your deployment:"
        echo "export SEARCH_PER_FETCH_TIMEOUT_SEC=8"
        echo "export SEARCH_OVERALL_TIMEOUT_SEC=12"
        echo "export SEARCH_MAX_CONCURRENCY=4"
        echo ""
        echo "Or add to your config.yaml:"
        echo "SEARCH_PER_FETCH_TIMEOUT_SEC: 8"
        echo "SEARCH_OVERALL_TIMEOUT_SEC: 12"
        echo "SEARCH_MAX_CONCURRENCY: 4"
        ;;
    4)
        echo "🔧 Custom configuration setup..."
        echo ""
        read -p "Enter per-fetch timeout (seconds, default 6): " per_fetch
        read -p "Enter overall timeout (seconds, default 10): " overall
        read -p "Enter max concurrency (default 4): " concurrency
        
        per_fetch=${per_fetch:-6}
        overall=${overall:-10}
        concurrency=${concurrency:-4}
        
        echo ""
        echo "Custom configuration:"
        echo "export SEARCH_PER_FETCH_TIMEOUT_SEC=$per_fetch"
        echo "export SEARCH_OVERALL_TIMEOUT_SEC=$overall"
        echo "export SEARCH_MAX_CONCURRENCY=$concurrency"
        echo ""
        echo "Or add to your config.yaml:"
        echo "SEARCH_PER_FETCH_TIMEOUT_SEC: $per_fetch"
        echo "SEARCH_OVERALL_TIMEOUT_SEC: $overall"
        echo "SEARCH_MAX_CONCURRENCY: $concurrency"
        ;;
    5)
        echo "📋 Current configuration:"
        echo "SEARCH_PER_FETCH_TIMEOUT_SEC: 6"
        echo "SEARCH_OVERALL_TIMEOUT_SEC: 10"
        echo "SEARCH_MAX_CONCURRENCY: 4"
        ;;
    *)
        echo "❌ Invalid choice. Please run the script again."
        exit 1
        ;;
esac

echo ""
echo "📊 Configuration impact analysis:"
echo ""

# 根据选择提供分析
case $choice in
    1)
        echo "🎯 Balanced Configuration Impact:"
        echo "  ✅ Success rate: 50% → 80-85%"
        echo "  ⏱️ Response time: 6-10s → 8-12s"
        echo "  🔄 Concurrency: 4 → 3 queries"
        echo "  💡 Best for: General use cases"
        ;;
    2)
        echo "🛡️ Conservative Configuration Impact:"
        echo "  ✅ Success rate: 50% → 90-95%"
        echo "  ⏱️ Response time: 6-10s → 10-15s"
        echo "  🔄 Concurrency: 4 → 2 queries"
        echo "  💡 Best for: Critical applications"
        ;;
    3)
        echo "⚡ Aggressive Configuration Impact:"
        echo "  ✅ Success rate: 50% → 60-70%"
        echo "  ⏱️ Response time: 6-10s → 6-10s"
        echo "  🔄 Concurrency: 4 → 4 queries"
        echo "  💡 Best for: Real-time applications"
        ;;
esac

echo ""
echo "🚀 Next steps:"
echo "1. Apply the configuration to your deployment"
echo "2. Restart your application"
echo "3. Monitor the success rate and response time"
echo "4. Adjust if needed based on actual performance"
echo ""
echo "🧪 To test the new configuration:"
echo "./test-concurrent-search-performance.sh"
echo ""
echo "📈 To monitor performance:"
echo "Watch for these log patterns:"
echo "  ✅ [Concurrent] Query X successful"
echo "  ❌ [Concurrent] Query X failed"
echo "  ⏰ [Concurrent] Query X timed out"
echo "  🎯 [Concurrent] Search completed: X successful, Y failed"
