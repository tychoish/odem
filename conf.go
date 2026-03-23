package shbot

import "github.com/tychoish/grip/level"

type Configuration struct {
	Level level.Priority `bson:"log_level" json:"log_level" yaml:"log_level"`
}
