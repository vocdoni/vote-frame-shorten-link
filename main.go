package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"lukechampine.com/blake3"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var allowedDomains []string
var client *mongo.Client
var urlCollection *mongo.Collection

type URLMapping struct {
	ShortLink string `bson:"shortLink"`
	LongLink  string `bson:"longLink"`
}

type URLMappingResponse struct {
	ShortLink string `json:"link"`
}

func init() {
	if os.Getenv("ALLOWED_DOMAINS") == "" || os.Getenv("MONGO_URI") == "" || os.Getenv("MONGO_DB") == "" {
		log.Fatal("Environment variables not set, please set ALLOWED_DOMAINS, MONGO_URI and MONGO_DB")
	}

	allowedDomains = strings.Split(os.Getenv("ALLOWED_DOMAINS"), ",")

	var err error
	opts := options.Client()
	opts.ApplyURI(os.Getenv("MONGO_URI"))
	opts.SetMaxConnecting(100)
	timeout := time.Second * 10
	opts.ConnectTimeout = &timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, err = mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	urlCollection = client.Database(os.Getenv("MONGO_DB")).Collection("urls")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = urlCollection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.M{"shortLink": 1}, // Create an index on the shortLink field
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

func addURLHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request: %s\n", r.URL.Path)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	domain := parts[2]
	longLink := strings.Join(parts[3:], "/")

	if !isDomainAllowed(domain) {
		http.Error(w, "Domain not allowed", http.StatusBadRequest)
		return
	}

	// search in longLink the Vocdoni processID
	// if found, then add the processID to the shortLink
	processID := []byte{}
	for _, l := range parts[3:] {
		if len(l) == 64 {
			if b, err := hex.DecodeString(l); err == nil {
				processID = b
				break
			}
		}
	}
	shortLink := uuid.New().String()[:8]
	shortLinkType := "uuid"
	if processID != nil {
		// We take the 8 first chars of the base64 encoded hash of the processID.
		// The probability of at least one collision among 100,000 generated hashes is approximately 0.0018%.
		// The probability of at least one collision among 1,000,000 generated hashes is approximately 0.177%.
		hash := blake3.Sum256(processID)
		shortLink = base64.StdEncoding.EncodeToString(hash[:])[:8]
		shortLinkType = "processID"
	}

	mapping := URLMapping{
		ShortLink: shortLink,
		LongLink:  fmt.Sprintf("https://%s/%s", domain, longLink),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := urlCollection.InsertOne(ctx, mapping)
	if err != nil {
		http.Error(w, "Error saving to database", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(URLMappingResponse{ShortLink: shortLink})
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	fmt.Printf("new link type %s %s => %s\n", shortLinkType, shortLink, mapping.LongLink)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		log.Println(err.Error())
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortLink := strings.TrimPrefix(r.URL.Path, "/")
	if len(shortLink) < 8 {
		http.Redirect(w, r, "https://farcaster.vote", http.StatusFound)
		return
	}
	var mapping URLMapping
	err := urlCollection.FindOne(context.Background(), bson.M{"shortLink": shortLink}).Decode(&mapping)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, mapping.LongLink, http.StatusFound)
}

func isDomainAllowed(domain string) bool {
	for _, d := range allowedDomains {
		if d == domain {
			return true
		}
	}
	return false
}

func main() {
	fmt.Printf("Allowed domains: %v\n", allowedDomains)
	fmt.Printf("MongoDB URI: %s\n", os.Getenv("MONGO_URI"))
	fmt.Printf("MongoDB DB: %s\n", os.Getenv("MONGO_DB"))
	fmt.Println("Server started")
	http.HandleFunc("/add/", addURLHandler)
	http.HandleFunc("/", redirectHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
