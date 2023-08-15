package context

import (
	"context"
	"log"
	"net"
	"encoding/binary"
	"time"
	"math"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/free5gc/smf/internal/logger"
)

type dhcpKey struct {
	Selection *UPFSelectionParams `bson:"selection"`
	Supi      string              `bson:"SUPI"`
}

type dhcpValue struct {
	Addr net.IP `bson:"IP_byte"`
}

// MongoDB constants
const (
	mongoURI        = "mongodb://localhost:27017"
	mongoDBName     = "free5gc"
	mongoCollection = "DHCPMemory"
)

var (
	mongoClient *mongo.Client
	dhcpCollection *mongo.Collection
)

func init() {
	// Initialize MongoDB client
	var err error
	mongoClient, err = mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB server: %v", err)
	}

	// Get the DHCPMemory collection
	dhcpCollection = mongoClient.Database(mongoDBName).Collection(mongoCollection)
}

func DHCPCheck(ipVal net.IP) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mongoClient.Ping(ctx, nil)
	if err != nil {
		return false
	}

	filter := bson.M{
		"IP Address": ipVal.String(),
	}

	count, err := dhcpCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false
	}

	return count > 0
}

func GetFromDHCPMemory(cacheKey dhcpKey) (dhcpValue, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{
        "SUPI":         cacheKey.Supi,
        "UPFSelection": cacheKey.Selection,
    }

    var result bson.M
    err := dhcpCollection.FindOne(ctx, filter).Decode(&result)
    if err != nil {
        return dhcpValue{}, err
    }

    dhcpAddr := result["IP Address"].(string)
    ipVal := net.ParseIP(dhcpAddr)
    buf := make([]byte, 4)
    octetValue := ipVal.To4()

    var intValue int
    for i := 0; i < 4; i++ {
        intValue += int(octetValue[i]) * int(math.Pow(256, float64(3-i)))
    }
    binary.BigEndian.PutUint32(buf, uint32(intValue))

    dhcpVal := dhcpValue{
        Addr: buf,
    }

    // Renew the expiration date
    expirationDate := time.Now().Add(24 * time.Hour)
    update := bson.M{
        "$set": bson.M{
            "expirationDate": expirationDate,
        },
    }
    _, err = dhcpCollection.UpdateOne(ctx, filter, update)
    if err != nil {
        return dhcpValue{}, err
    }

    return dhcpVal, nil
}

func SetToDHCPMemory(cacheKey dhcpKey, cacheValue dhcpValue) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"SUPI":          cacheKey.Supi,
		"UPFSelection":  cacheKey.Selection,
		"IP Address":    cacheValue.Addr.String(),
	}

	var result bson.M
	err := dhcpCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Fatalf("Failed to query MongoDB collection: %v", err)
	}

	expirationDate := time.Now().Add(24 * time.Hour)
	// DHCPCheck(cacheValue.Addr.String())
	if result == nil {
		doc := bson.M{
			"SUPI":           cacheKey.Supi,
			"UPFSelection":   cacheKey.Selection,
			"IP Address":     cacheValue.Addr.String(),
			"expirationDate": expirationDate,
		}

		logger.CtxLog.Infof("%v", cacheValue.Addr)

		_, err := dhcpCollection.InsertOne(ctx, doc)
		if err != nil {
			log.Fatalf("Failed to insert document into MongoDB collection: %v", err)
		}
	}
}