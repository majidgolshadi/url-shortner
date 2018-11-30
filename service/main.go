package main

import (
	"github.com/majidgolshadi/url-shortner"
	"log"
	"time"
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

	startTime := time.Now()
	println(url_shortner.NewUrl(";drop table map_url;"))
	println(time.Since(startTime)/time.Millisecond)

	//tk, err := url_shortner.NewUrlWithCustomToken("http://google.com", "cc")
	//if err != nil {
	//	println(err.Error())
	//	println(tk)
	//}
	//
	//tk1, err1 := url_shortner.NewUrlWithCustomToken("http://google.com/8aa", "nb")
	//if err != nil {
	//	println(err1.Error())
	//	println(tk1)
	//}
	//
	//println("without error", tk1)
	//
	//
	//url_shortner.RunRestApi(":9001")
}
