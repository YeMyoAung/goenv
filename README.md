# go-env-loader

A simple Go package to load environment variables from a `.env` file or the current environment into a struct, with validation support.

## Installation

```
go get github.com/YeMyoAung/goenv
```

## Usage

### 1. Define your config struct

```go
import "github.com/YeMyoAung/goenv"

type Config struct {
    Name      string            `json:"name" validate:"required"`
    Foods     []string          `json:"foods" validate:"required"`
    Age       int               `json:"age" validate:"required"`
    IsStudent bool              `json:"is_student"`
    Height    float64           `json:"height" validate:"required"`
    TTL       time.Duration     `json:"ttl" validate:"required"`
    Headers   map[string]string `json:"headers" validate:"required"`
}
```
### 2. Load from environment variables

```go
config, err := goenv.Load[Config](nil)
if err != nil {
    // handle error
}
fmt.Println(config)
```

### 3. Load from a .env file

```go
args := &goenv.Args{
    FileName: ".env",
}
config, err := goenv.Load[Config](args)
if err != nil {
    // handle error
}
fmt.Println(config)
```

### 4. Validation

By default, all struct fields with the `validate:"required"` tag are validated. You can provide a custom validator via `Args` if needed.

## Example .env file

```
name=John
foods=[apple,banana,orange]
age=30
is_student=true
height=1.75
ttl=1200s
headers={"Authorization":"Bearer token","X-App":"klink"}
```

## License

MIT
