package main

import (
	"time"

	"github.com/lomik/zapwriter"
	"github.com/lunny/html2md"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
	"math/rand"
)

type UpdateType int

const (
	NewRelease UpdateType = iota
	Retag
	DescriptionChange
)

type Update struct {
	Type   UpdateType
	Repo   string
	Filter string

	Title   string
	Content string
	Link    string
}

type Feed struct {
	Id             int
	Repo           string
	Filter         string
	Name           string
	MessagePattern string

	lastUpdateTime time.Time
	logger         *zap.Logger
	cfg            FeedsConfig
}

func NewFeed(id int, repo, filter, name, messagePattern string) (*Feed, error) {
	return &Feed{
		Id:             id,
		Repo:           repo,
		Filter:         filter,
		Name:           name,
		MessagePattern: messagePattern,

		lastUpdateTime: time.Unix(0, 0),

		logger: zapwriter.Logger("main").With(
			zap.String("feed_repo", repo),
			zap.Int("id", id),
		),
	}, nil
}

func (f *Feed) SetCfg(cfg FeedsConfig) {
	f.cfg = cfg
}

func (f *Feed) ProcessFeed() {
	cfg := f.cfg

	url := "https://github.com/" + f.Repo + "/releases.atom"

	// Initialize
	for i := range cfg.Filters {
		cfg.Filters[i].lastUpdateTime = getLastUpdateTime(url, cfg.Filters[i].Filter)
	}

	fp := gofeed.NewParser()
	if cfg.PollingInterval == 0 {
		cfg.PollingInterval = 15 * time.Minute
	}

	delay := time.Duration(rand.Int()) % cfg.PollingInterval
	t0 := time.Now()
	nextRun := t0.Add(delay)
	f.logger.Info("will process feed",
		zap.Duration("extra_delay", delay),
		zap.Time("nextRun", nextRun),
	)
	for {
		dt := time.Until(nextRun)
		if dt > 0 {
			time.Sleep(dt)
		}
		nextRun = nextRun.Add(cfg.PollingInterval)
		t0 = time.Now()
		for i := range cfg.Filters {
			cfg.Filters[i].filterProcessed = false
		}

		feed, err := fp.ParseURL(url)
		if err != nil {
			f.logger.Info("done", zap.Duration("runtime", time.Since(t0)),
				zap.Duration("runtime", time.Since(t0)),
				zap.Time("nextRun", nextRun),
				zap.Time("now", t0),
				zap.Error(err),
			)
			continue
		}

		processedFilters := 0
		for _, item := range feed.Items {
			f.logger.Debug("processing item",
				zap.String("title", item.Title),
			)
			for i := range cfg.Filters {
				if cfg.Filters[i].lastUpdateTime.Unix() >= item.UpdatedParsed.Unix() {
					cfg.Filters[i].filterProcessed = true
				}
				if cfg.Filters[i].filterProcessed {
					f.logger.Debug("item already processed by this filter",
						zap.String("title", item.Title),
						zap.String("filter", cfg.Filters[i].Filter),
					)
					continue
				}
				f.logger.Debug("testing for filter",
					zap.String("filter", cfg.Filters[i].Filter),
				)
				if cfg.Filters[i].filterRegex.MatchString(item.Title) {
					notification := cfg.Repo + " was tagged: " + item.Title + "\nLink: " + item.Link

					if len(item.Content) != 0 && item.Content != item.Title {
						notification += "\nRelease notes:\n" + html2md.Convert(item.Content)
					}

					f.logger.Info("release tagged",
						zap.String("release", item.Title),
						zap.String("notification", notification),
					)

					methods, err := getNotificationMethods(cfg.Repo, cfg.Filters[i].Name)
					if err != nil {
						f.logger.Info("error",
							zap.Error(err),
						)
					} else {
						f.logger.Info("notifications",
							zap.Strings("methods", methods),
						)
						for _, m := range methods {
							config.senders[m].Send(cfg.Repo, cfg.Filters[i].Name, notification)
						}
					}

					cfg.Filters[i].filterProcessed = true
					cfg.Filters[i].lastUpdateTime = *item.UpdatedParsed
					updateLastUpdateTime(url, cfg.Filters[i].Filter, cfg.Filters[i].lastUpdateTime)
				}
			}

			if len(cfg.Filters) == processedFilters {
				break
			}
		}
		f.logger.Info("done",
			zap.Duration("runtime", time.Since(t0)),
			zap.Time("nextRun", nextRun),
			zap.Time("now", t0),
		)
	}
}
