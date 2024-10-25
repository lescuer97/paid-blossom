package core

import (
	"ratasker/internal/database"
	"ratasker/internal/io"
)

type BlossomServer struct {
	DB database.Database
	IO io.BlossomIO
}
