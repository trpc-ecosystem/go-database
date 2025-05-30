package main

import (
	"trpc.group/trpc-go/trpc-database/gorm"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// User is the model struct.
type User struct {
	ID       int
	Username string
}

func main() {
	_ = trpc.NewServer()

	cli, err := gorm.NewClientProxy("trpc.clickhouse.server.service")
	if err != nil {
		panic(err)
	}

	// Create record
	insertUser := User{Username: "gorm-client"}
	if result := cli.Create(&insertUser); result.Error != nil {
		panic(result.Error)
	}
	log.Infof("inserted data succeed")

	// Query record
	var queryUser User
	if err := cli.First(&queryUser).Error; err != nil {
		panic(err)
	}
	log.Infof("query user: %+v", queryUser)

	// Delete record
	deleteUser := User{}
	if err := cli.Where("username = ?", "gorm-client").Delete(&deleteUser).Error; err != nil {
		panic(err)
	}
	log.Info("delete record succeed")

	// For more use cases, see https://gorm.io/docs/create.html
}
