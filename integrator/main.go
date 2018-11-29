package main

import (
	"github.com/majidgolshadi/url-shortner"
	"log"
)

func main() {

	config := &url_shortner.EtcdConfig{
		Hosts: []string{"http://127.0.0.1:2379"},
		RootKey: "/service",
		NodeId: "node1",
	}

	dbConfig := &url_shortner.MariaDbConfig{
		Host: "127.0.0.1:3306",
		Username: "root",
		Password: "123",
		Database: "tiny_url",
	}

	err := url_shortner.InitTokenGenerator(config, dbConfig)
	if err != nil {
		log.Fatal(err.Error())
	}


	for i:=1; i < 1099; i++ {
		token := url_shortner.NewUrl("http://google.com/man")
		println(token)
	}
}
