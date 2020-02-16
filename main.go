package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-redis/redis/v7"
)

type payload struct {
	Addr string   `json:"addr"`
	DB   int      `json:"db"`
	Cmd  []string `json:"cmd"`
}

func main() {

	http.HandleFunc("/api/cmd", handlerFunc)

	http.ListenAndServe("0.0.0.0:1337", nil)
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if r.Method != "POST" {
		abort(w, "Cannot "+r.Method)
		return
	}

	p, err := parsePayload(r.Body)
	if err != nil {
		abort(w, err.Error())
		return
	}

	client, err := initRedisClient(p.Addr, p.DB)

	if err != nil {
		abort(w, "Cannot connect to Redis")
		return
	}

	result, err := processCommand(client, p.Cmd)

	if err != nil {
		fmt.Println(err)
	}

	respondJSON(w, result)
	client.Close()
}

func abort(w http.ResponseWriter, err string) {
	w.WriteHeader(http.StatusBadRequest)

	j, _ := json.Marshal(map[string]string{
		"error": err,
	})

	fmt.Fprint(w, string(j))
}

func parsePayload(b io.ReadCloser) (payload, error) {
	p := new(payload)
	jsonPayload, err := ioutil.ReadAll(b)
	if err != nil {
		return payload{}, err
	}

	err = json.Unmarshal([]byte(jsonPayload), &p)
	if err != nil {
		return payload{}, err
	}

	if len(p.Cmd) == 0 {
		return payload{}, errors.New("Empty payload")
	}

	return *p, nil
}

func initRedisClient(addr string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	return client, nil
}

func processCommand(client *redis.Client, command []string) (interface{}, error) {
	args := stringArrayToInterfaceArray(command)
	cmd := redis.NewCmd(args...)
	client.Process(cmd)

	return cmd.Result()
}

func stringArrayToInterfaceArray(s []string) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

func respond(w http.ResponseWriter, statusCode int, body string) {
	w.WriteHeader(statusCode)
	fmt.Fprint(w, body)
}

func respondJSON(w http.ResponseWriter, result interface{}) {
	data := map[string]interface{}{
		"result": result,
	}
	j, _ := json.Marshal(data)

	respond(w, http.StatusOK, string(j))
}
