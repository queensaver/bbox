module github.com/queensaver/bbox/server

go 1.15

require (
	github.com/google/go-cmp v0.5.5
	github.com/queensaver/openapi/golang/proto/services v0.0.0-20220321063112-2e0f4b99d7d8
	github.com/queensaver/packages/config v0.0.0-20210929054635-954a86633ecc
	github.com/queensaver/packages/logger v0.0.0-20210930150643-4a50b289ebea
	github.com/queensaver/packages/scale v0.0.0-20220404192636-0b664c8252b7
	github.com/queensaver/packages/sound v0.0.0-20220404192636-0b664c8252b7
	github.com/queensaver/packages/temperature v0.0.0-20220404192636-0b664c8252b7
	github.com/queensaver/packages/varroa v0.0.0-20220404192636-0b664c8252b7
	github.com/robfig/cron/v3 v3.0.1
	github.com/stianeikeland/go-rpio/v4 v4.4.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	periph.io/x/conn/v3 v3.6.8
	periph.io/x/host/v3 v3.7.0
)
