package main

import (
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	signaling "streaming-signaling.jounetsism.biz/proto/signaling"
)

func main() {
	// Ginエンジンのインスタンスを作成
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	InitFirebase()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})

	wsAuth := r.Group("/ws")
	wsAuth.Use(FirebaseWebsocketAuth())
	wsAuth.GET("/live/:roomId", websocketHandler)
	wsAuth.GET("/live/:roomId/viewer", websocketViewerHandler)
	// wsAuth.GET("/live/:roomId/viewer", websocketViewerHandler)

	httpAuth := r.Group("/")
	httpAuth.Use(AuthHttpInterceptor())
	go r.Run(":8080")
	lis, _ := net.Listen("tcp", ":50051")

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(AuthGrpcInterceptor),)

	signaling.RegisterSignalingServiceServer(
		grpcServer,
		&SignalingService{},
	)
	logrus.Info("gRPC server started on :50051")
	err := grpcServer.Serve(lis)
	if err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}

}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = &Rooms{
	map[string]*Room{},
	sync.RWMutex{},
}

