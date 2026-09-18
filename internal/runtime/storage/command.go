package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
)

func ValidCommand(args []string) bool {
	if len(args) == 1 && args[0] == "buckets" {
		return true
	}
	if len(args) == 2 && (args[0] == "create-bucket" || args[0] == "list" || args[0] == "put") {
		return bucketName.MatchString(args[1])
	}
	return len(args) == 3 && (args[0] == "get" || args[0] == "delete") && bucketName.MatchString(args[1]) && objectID.MatchString(args[2])
}

const Usage = "usage: basestack storage buckets | create-bucket <bucket> | list <bucket> | put <bucket> (stdin) | get <bucket> <id> (stdout) | delete <bucket> <id>"

func Command(ctx context.Context, store Store, args []string, in io.Reader, out io.Writer) error {
	if !ValidCommand(args) {
		return errors.New(Usage)
	}
	var data any
	var err error
	switch args[0] {
	case "buckets":
		data, err = store.Buckets(ctx)
	case "create-bucket":
		err = store.CreateBucket(ctx, args[1])
		data = map[string]bool{"created": err == nil}
	case "list":
		data, err = store.List(ctx, args[1])
	case "put":
		data, err = store.Put(ctx, args[1], in)
	case "get":
		var f io.ReadCloser
		f, _, err = store.Open(ctx, args[1], args[2])
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(out, f)
		return err
	case "delete":
		err = store.Delete(ctx, args[1], args[2])
		data = map[string]bool{"deleted": err == nil}
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(data)
}
