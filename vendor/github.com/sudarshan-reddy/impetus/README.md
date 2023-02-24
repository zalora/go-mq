# impetus
impetus is a very light wrapper over time.Ticker that causes the first tick to 
be instant

## Usage :

The usage is very simple and the same as using a time.Ticker. 

```go
    ticker  := impetus.NewImmediateTicker(5 * time.Seconds)
    for range ticker.C{
        fmt.Println("tick doesn't wait for 5 seconds")
    }
```
