package database

import (
	"runtime"
	"strings"
)

// PoolProfile selects a connection-pool sizing formula.
type PoolProfile string

const (
	// PoolConservative keeps database pressure low for small apps and shared DBs.
	PoolConservative PoolProfile = "conservative"
	// PoolBalanced is a general-purpose profile for mixed CPU and IO work.
	PoolBalanced PoolProfile = "balanced"
	// PoolIOHeavy allows more in-flight queries when the app is often waiting
	// on network/storage IO and the database can absorb the extra concurrency.
	PoolIOHeavy PoolProfile = "io-heavy"
)

// PoolStrategy describes how to derive pool sizes from CPU and IO capacity.
type PoolStrategy struct {
	// Profile selects the sizing formula. Unknown values fall back to balanced.
	Profile PoolProfile
	// CPUCores is the effective CPU parallelism. Values <= 0 use GOMAXPROCS.
	CPUCores int
	// IOParallelism represents extra independent IO capacity, such as separate
	// disks, database replicas, or known downstream query capacity.
	IOParallelism int
	// MaxOpenLimit caps the derived MaxOpenConns. Values <= 0 use the profile default.
	MaxOpenLimit int
}

// RuntimePoolStrategy returns a strategy using the process CPU parallelism.
func RuntimePoolStrategy(profile PoolProfile) PoolStrategy {
	return PoolStrategy{
		Profile:       profile,
		CPUCores:      runtime.GOMAXPROCS(0),
		IOParallelism: 1,
	}
}

// ApplyPoolStrategy returns opts with MaxOpenConns and MaxIdleConns derived
// from strategy. Other Options fields are preserved.
func ApplyPoolStrategy(opts Options, strategy PoolStrategy) Options {
	profile := normalizePoolProfile(strategy.Profile)
	cores := strategy.CPUCores
	if cores <= 0 {
		cores = runtime.GOMAXPROCS(0)
	}
	if cores < 1 {
		cores = 1
	}
	ioParallelism := strategy.IOParallelism
	if ioParallelism < 0 {
		ioParallelism = 0
	}

	var maxOpen, defaultLimit int
	var idleRatio int
	switch profile {
	case PoolConservative:
		maxOpen = cores + ioParallelism
		defaultLimit = 16
		idleRatio = 4
	case PoolIOHeavy:
		maxOpen = cores*4 + ioParallelism*2
		defaultLimit = 128
		idleRatio = 2
	default:
		maxOpen = cores*2 + ioParallelism
		defaultLimit = 64
		idleRatio = 3
	}

	if maxOpen < 1 {
		maxOpen = 1
	}
	limit := strategy.MaxOpenLimit
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxOpen > limit {
		maxOpen = limit
	}

	maxIdle := maxOpen / idleRatio
	if maxIdle < 1 {
		maxIdle = 1
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	opts.MaxOpenConns = maxOpen
	opts.MaxIdleConns = maxIdle
	return opts
}

func normalizePoolProfile(profile PoolProfile) PoolProfile {
	switch PoolProfile(strings.ToLower(strings.TrimSpace(string(profile)))) {
	case PoolConservative:
		return PoolConservative
	case PoolIOHeavy:
		return PoolIOHeavy
	case PoolBalanced:
		return PoolBalanced
	default:
		return PoolBalanced
	}
}
