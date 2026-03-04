module github.com/MyroslavMokhammadAbdeljawwad/emoji-meta-api

go 1.22

require (
	github.com/gin-gonic/gin v1.10.3
	github.com/golang-jwt/jwt/v5 v5.0.1
	github.com/go-redis/cache/v8 v8.2.1
	github.com/go-redis/redis/v8 v8.16.4
	github.com/patrickmn/go-cache v2.2.0
	github.com/spf13/cobra v1.7.3
	github.com/spf13/viper v1.18.0
	github.com/stretchr/testify v1.12.5
	google.golang.org/grpc v1.67.0
)

replace (
	github.com/go-redis/cache/v8 => github.com/go-redis/cache/v8 v8.2.1
)