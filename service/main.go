package main

import (
	"github.com/majidgolshadi/url-shortner"
	"log"
	"time"
)

func main() {

	coordinator, err := url_shortner.NewEtcd(&url_shortner.EtcdConfig{
		Hosts: []string{"http://127.0.0.1:2379"},
		RootKey: "/service",
		NodeId: "node1",
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	counter, err := url_shortner.NewDistributedAtomicCounter(coordinator)
	if err != nil {
		log.Fatal(err.Error())
	}

	db ,err := url_shortner.DbConnect(&url_shortner.MariaDbConfig{
		Host: "127.0.0.1:3306",
		Username: "root",
		Password: "123",
		Database: "tiny_url",
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	tg := url_shortner.NewTokenGenerator(counter, db)

	if err != nil {
		log.Fatal(err.Error())
	}

	startTime := time.Now()
	println(tg.NewUrl("http://google.com/new_url"))
	println(time.Since(startTime)/time.Millisecond)

	tk, err := tg.NewUrlWithCustomToken("http://google.com", "cc")
	if err != nil {
		println(err.Error())
		println(tk)
	}

	tk1, err1 := tg.NewUrlWithCustomToken("http://google.com/8aa", "nb")
	if err != nil {
		println(err1.Error())
		println(tk1)
	}

	println("without error", tk1)


	url_shortner.RunRestApi(tg, db, "secret key",":9001")
}
