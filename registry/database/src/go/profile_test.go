package database

import "testing"

func TestApplyPoolStrategyBalanced(t *testing.T) {
	opts := ApplyPoolStrategy(Options{}, PoolStrategy{
		Profile:       PoolBalanced,
		CPUCores:      4,
		IOParallelism: 2,
	})
	if opts.MaxOpenConns != 10 {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != 3 {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
}

func TestApplyPoolStrategyIOHeavyUsesExtraCapacity(t *testing.T) {
	opts := ApplyPoolStrategy(Options{}, PoolStrategy{
		Profile:       PoolIOHeavy,
		CPUCores:      4,
		IOParallelism: 4,
	})
	if opts.MaxOpenConns != 24 {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != 12 {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
}

func TestApplyPoolStrategyLimit(t *testing.T) {
	opts := ApplyPoolStrategy(Options{}, PoolStrategy{
		Profile:       PoolIOHeavy,
		CPUCores:      32,
		IOParallelism: 32,
		MaxOpenLimit:  20,
	})
	if opts.MaxOpenConns != 20 {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != 10 {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
}

func TestFromEnvPoolProfile(t *testing.T) {
	t.Setenv("DATABASE_POOL_PROFILE", "io-heavy")
	t.Setenv("DATABASE_POOL_CPU_CORES", "4")
	t.Setenv("DATABASE_IO_PARALLELISM", "4")
	t.Setenv("DATABASE_POOL_MAX_OPEN_LIMIT", "30")

	opts := FromEnv()
	if opts.MaxOpenConns != 24 || opts.MaxIdleConns != 12 {
		t.Fatalf("pool profile not applied: %#v", opts)
	}
}

func TestFromEnvPoolKnobsWithoutProfile(t *testing.T) {
	t.Setenv("DATABASE_POOL_CPU_CORES", "4")
	t.Setenv("DATABASE_IO_PARALLELISM", "3")
	t.Setenv("DATABASE_POOL_MAX_OPEN_LIMIT", "20")

	opts := FromEnv()
	if opts.MaxOpenConns != 11 || opts.MaxIdleConns != 3 {
		t.Fatalf("pool knobs not applied: %#v", opts)
	}
}

func TestFromEnvExplicitPoolOverridesProfile(t *testing.T) {
	t.Setenv("DATABASE_POOL_PROFILE", "io-heavy")
	t.Setenv("DATABASE_POOL_CPU_CORES", "4")
	t.Setenv("DATABASE_IO_PARALLELISM", "4")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "9")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "2")

	opts := FromEnv()
	if opts.MaxOpenConns != 9 || opts.MaxIdleConns != 2 {
		t.Fatalf("explicit pool values should win: %#v", opts)
	}
}
