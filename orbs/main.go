package main

import (
	"net/http"
	//"github.com/schollz/httpfileserver"
	"log"
	"os"
	"orbs/orbserver"
	"strconv"
	"io/ioutil"
	"encoding/json"
)

var (
	res_index_path = "public/play/gamesdefault/index.json"
	NUM_ROOMS = 500 //!!! change this if not hosting yume nikki
)

func main() {
	delimchar := "\uffff";
	log.Println("test" + delimchar + "test")

	port := os.Getenv("PORT")

	if (port == "") {
		//log.Fatal("$PORT must be set")
		port = "8080"
	}

	res_index_data, err := ioutil.ReadFile(res_index_path)
	if err != nil {
		log.Fatal(err)
	}

	var res_index interface{}

	err = json.Unmarshal(res_index_data, &res_index)
	if err != nil {
		log.Fatal(err)
	}

	cache, ok := res_index.(map[string]interface{})
	if !ok {
		log.Fatal("could not convert cache to map[string]interface{}")
	}

	charset, ok := cache["cache"].(map[string]interface{})
	if !ok {
		log.Fatal("Could not convert charset to map[string]interface{}")
	}

	vals, ok := charset["charset"].(map[string]interface{})
	if !ok {
		log.Fatal("Could not convert vals to map[string]interface{}")
	}

	//list of valid game character sprite resource keys
	var spriteNames []string
	for k := range vals {
		if k != "_dirname" {
			spriteNames = append(spriteNames, k)
		}
	}

	var roomNames []string

	for i:=0; i < NUM_ROOMS; i++ {
		roomNames = append(roomNames, strconv.Itoa(i))
	}

	for name := range roomNames {
		hub := orbserver.NewHub(roomNames[name], spriteNames)
		go hub.Run()
	}

	//http.Handle("/", httpfileserver.New("/", "public/"))
	http.Handle("/", http.FileServer(http.Dir("public/")))
	//http.HandleFunc("/", Handler)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}

/*func Handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}*/
