package main

import "fmt"

func main() {
    params := map[string]interface{}{
        "current": 1,
        "pageSize": 10,
        "deviceTypeName": "台式机",  // 注意大小写
    }
    
    fmt.Println("Map keys and values:")
    for k, v := range params {
        fmt.Printf("  [%s] = %v (type: %T)\n", k, v, v)
    }
    
    // 测试类型断言
    if val, ok := params["deviceTypeName"].(string); ok {
        fmt.Printf("\n✓ params[\"deviceTypeName\"] found: %s\n", val)
    } else {
        fmt.Println("\n✗ params[\"deviceTypeName\"] not found or wrong type")
    }
}
