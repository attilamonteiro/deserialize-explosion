package model

import "strconv"

// Shared models and small helpers

type Recipe struct {
    ID    int               `json:"id"`
    Title string            `json:"title"`
    Info  Info              `json:"info"`
    Steps []Step            `json:"steps"`
    Meta  map[string]string `json:"meta"`
}

type Info struct {
    Calories float64 `json:"calories"`
    Notes    string  `json:"notes"`
}

type Step struct {
    Index       int    `json:"index"`
    Description string `json:"description"`
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func AtoiOrDefault(s string, def int) int {
    if s == "" {
        return def
    }
    v, err := strconv.Atoi(s)
    if err != nil {
        return def
    }
    return v
}
