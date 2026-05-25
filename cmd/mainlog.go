package main

import (
    _ "github.com/ruoyi-next/ruoyi-go-backend/internal/server"
)

func init() {
    // This forces the server package to be linked
    println("DEBUG: server package linked, embed tag should be active")
}
