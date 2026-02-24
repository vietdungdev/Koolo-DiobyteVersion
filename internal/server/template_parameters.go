package server

import (
	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/koolo/internal/bot"
	"github.com/hectorgimenez/koolo/internal/config"
)

type TZGroup struct {
	Act           int
	Name          string
	PrimaryAreaID int
	Immunities    []string
	BossPacks     string
	ExpTier       string
	LootTier      string
}

// SchedulerStatusInfo contains scheduler state for UI display
type SchedulerStatusInfo struct {
	Enabled        bool             `json:"enabled"`
	Mode           string           `json:"mode"`
	Phase          string           `json:"phase"`
	PhaseStartTime string           `json:"phaseStartTime"`
	PhaseEndTime   string           `json:"phaseEndTime"`
	TodayWakeTime  string           `json:"todayWakeTime"`
	TodayRestTime  string           `json:"todayRestTime"`
	PlayedMinutes  int              `json:"playedMinutes"`
	NextBreaks     []SchedulerBreak `json:"nextBreaks"`
	// WaitingForSchedule is true when the user clicked Play but the bot
	// is holding until the next scheduled window opens.
	WaitingForSchedule bool `json:"waitingForSchedule"`
	// ScheduledStartTime is the RFC3339 time the bot will auto-start.
	ScheduledStartTime string `json:"scheduledStartTime"`
	// Activated is true when the user has opted this character into scheduler
	// management by clicking Play. The scheduler is dormant until activated.
	Activated bool `json:"activated"`
	// ScheduleSummary is a human-readable description of the configured schedule
	// (e.g. "08:00–22:00" or "Duration: 14h play"). Shown in the dormant state
	// and the collapsed header so users know what the schedule is at a glance.
	ScheduleSummary string `json:"scheduleSummary"`
	// SimpleStopTime is the configured stop time for simple mode, surfaced so the
	// dashboard can display the full window (start–stop) in countdown displays.
	SimpleStopTime string `json:"simpleStopTime,omitempty"`
}

type SchedulerBreak struct {
	Type      string `json:"type"`
	StartTime string `json:"startTime"`
	Duration  int    `json:"duration"`
}

type IndexData struct {
	ErrorMessage                string
	Version                     string
	Status                      map[string]bot.Stats
	DropCount                   map[string]int
	AutoStart                   map[string]bool
	SchedulerStatus             map[string]*SchedulerStatusInfo `json:"schedulerStatus"`
	GlobalAutoStartEnabled      bool
	GlobalAutoStartDelaySeconds int
	ShowAutoStartPrompt         bool
}

type DropData struct {
	NumberOfDrops int
	Character     string
	Drops         []data.Drop
}

// AllDropsData is used by the centralized drops view.
type AllDropsData struct {
	ErrorMessage string
	Total        int
	Records      []AllDropRecord
}

// AllDropRecord flattens droplog.Record for templating.
type AllDropRecord struct {
	Time       string
	Supervisor string
	Character  string
	Profile    string
	Drop       data.Drop
}

type CharacterSettings struct {
	Version                 string
	ErrorMessage            string
	Supervisor              string
	CloneSource             string
	Config                  *config.CharacterCfg
	SkillOptions            []SkillOption
	SkillPrereqs            map[string][]string
	Saved                   bool
	DayNames                []string
	EnabledRuns             []string
	DisabledRuns            []string
	TerrorZoneGroups        []TZGroup
	RecipeList              []string
	RunewordRecipeList      []string
	RunewordFavoriteRecipes []string
	RunFavoriteRuns         []string
	RunewordRuneNames       map[string]string
	RunewordRerollable      map[string]bool
	AvailableProfiles       []string
	FarmerProfiles          []string
	LevelingSequenceFiles   []string
	Supervisors             []string
}

type SkillOption struct {
	Key  string
	Name string
}

type ConfigData struct {
	ErrorMessage   string
	CurrentVersion *VersionData
	*config.KooloCfg
}

type VersionData struct {
	CommitHash string
	CommitDate string
	CommitMsg  string
	Branch     string
}

type AutoSettings struct {
	ErrorMessage string
}
