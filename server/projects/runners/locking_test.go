package runners_test

import (
	"fmt"
	"testing"

	. "github.com/petergtz/pegomock"
	"github.com/runatlantis/atlantis/server/core/db"
	"github.com/runatlantis/atlantis/server/core/locking"
	"github.com/runatlantis/atlantis/server/events"
	eventsMocks "github.com/runatlantis/atlantis/server/events/mocks"
	"github.com/runatlantis/atlantis/server/events/mocks/matchers"
	eventsMatchers "github.com/runatlantis/atlantis/server/events/mocks/matchers"
	"github.com/runatlantis/atlantis/server/events/models"
	vcsmocks "github.com/runatlantis/atlantis/server/events/vcs/mocks"
	"github.com/runatlantis/atlantis/server/logging"
	"github.com/runatlantis/atlantis/server/projects/runners"
	"github.com/runatlantis/atlantis/server/projects/runners/mocks"
	. "github.com/runatlantis/atlantis/testing"
)

// Test that it runs the expected plan steps.
func TestLockingProjectCommandRunner_Plan(t *testing.T) {
	RegisterMockTestingT(t)

	t.Run("generate lock key url", func(t *testing.T) {
		mockLocker := eventsMocks.NewMockProjectLocker()
		mockProjectCommandRunner := mocks.NewMockProjectCommandRunner()

		runner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               mockLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		ctx := models.ProjectCommandContext{
			Log:        logging.NewNoopLogger(t),
			Workspace:  "default",
			RepoRelDir: ".",
		}

		When(mockLocker.TryLock(
			eventsMatchers.AnyPtrToLoggingSimpleLogger(),
			eventsMatchers.AnyModelsPullRequest(),
			eventsMatchers.AnyModelsUser(),
			AnyString(),
			eventsMatchers.AnyModelsProject(),
		)).ThenReturn(&events.TryLockResponse{
			LockAcquired: true,
			LockKey:      "lock-key",
		}, nil)

		When(mockProjectCommandRunner.Plan(
			matchers.EqModelsProjectCommandContext(ctx),
		)).ThenReturn(models.ProjectResult{
			Command:     models.PlanCommand,
			RepoRelDir:  ctx.RepoRelDir,
			Workspace:   ctx.Workspace,
			ProjectName: ctx.ProjectName,
			PlanSuccess: &models.PlanSuccess{},
		})

		res := runner.Plan(ctx)

		Equals(t, "https://lock-key", res.PlanSuccess.LockURL)
	})

	t.Run("prevent planning on locked projects", func(t *testing.T) {
		dbDir, cleanup := TempDir(t)
		defer cleanup()

		vcsClientMock := vcsmocks.NewMockClient()
		boltdb, _ := db.New(dbDir)

		projectLocker := &events.DefaultProjectLocker{
			Locker:    locking.NewClient(boltdb),
			VCSClient: vcsClientMock,
		}

		mockProjectCommandRunner := mocks.NewMockProjectCommandRunner()

		lockAcquiredRunner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               projectLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		thisCtx := models.ProjectCommandContext{
			Log:       logging.NewNoopLogger(t),
			Workspace: "default",
			Pull: models.PullRequest{
				BaseRepo: models.Repo{
					FullName: "locking-repo",
				},
				Num: 1,
			},
			RepoRelDir: ".",
		}

		When(mockProjectCommandRunner.Plan(
			matchers.EqModelsProjectCommandContext(thisCtx),
		)).ThenReturn(models.ProjectResult{
			Command:     models.PlanCommand,
			RepoRelDir:  thisCtx.RepoRelDir,
			Workspace:   thisCtx.Workspace,
			ProjectName: thisCtx.ProjectName,
			PlanSuccess: &models.PlanSuccess{},
		})

		res := lockAcquiredRunner.Plan(thisCtx)

		Assert(t, res.PlanSuccess != nil, "plan should be successful")
		mockProjectCommandRunner.VerifyWasCalledOnce().Plan(thisCtx)

		thatCtx := models.ProjectCommandContext{
			Log:       logging.NewNoopLogger(t),
			Workspace: "default",
			Pull: models.PullRequest{
				BaseRepo: models.Repo{
					FullName: "locking-repo",
				},
				Num: 2,
			},
			RepoRelDir: ".",
		}

		When(vcsClientMock.MarkdownPullLink(matchers.AnyModelsPullRequest())).ThenReturn("other PR", nil)

		mockProjectCommandRunner = mocks.NewMockProjectCommandRunner()
		lockFailedRunner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               projectLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		res = lockFailedRunner.Plan(thatCtx)

		Assert(t, res.PlanSuccess == nil, "plan success should be nil")
		Equals(t, res.Failure, "This project is currently locked by an unapplied plan from pull other PR. To continue, delete the lock from other PR or apply that plan and merge the pull request.\n\nOnce the lock is released, comment `atlantis plan` here to re-plan.")
	})
}

// Test that it runs the expected policy_check steps.
func TestLockingProjectCommandRunner_PolicyCheck(t *testing.T) {
	RegisterMockTestingT(t)

	t.Run("generate lock key url", func(t *testing.T) {
		mockLocker := eventsMocks.NewMockProjectLocker()
		mockProjectCommandRunner := mocks.NewMockProjectCommandRunner()

		runner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               mockLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		ctx := models.ProjectCommandContext{
			Log:        logging.NewNoopLogger(t),
			Workspace:  "default",
			RepoRelDir: ".",
		}

		When(mockLocker.TryLock(
			eventsMatchers.AnyPtrToLoggingSimpleLogger(),
			eventsMatchers.AnyModelsPullRequest(),
			eventsMatchers.AnyModelsUser(),
			AnyString(),
			eventsMatchers.AnyModelsProject(),
		)).ThenReturn(&events.TryLockResponse{
			LockAcquired: true,
			LockKey:      "lock-key",
		}, nil)

		When(mockProjectCommandRunner.PolicyCheck(
			matchers.EqModelsProjectCommandContext(ctx),
		)).ThenReturn(models.ProjectResult{
			Command:     models.PlanCommand,
			RepoRelDir:  ctx.RepoRelDir,
			Workspace:   ctx.Workspace,
			ProjectName: ctx.ProjectName,
			PlanSuccess: &models.PlanSuccess{},
		})

		res := runner.PolicyCheck(ctx)

		Equals(t, "https://lock-key", res.PlanSuccess.LockURL)
	})

	t.Run("prevent policy checking on locked projects", func(t *testing.T) {
		dbDir, cleanup := TempDir(t)
		defer cleanup()

		vcsClientMock := vcsmocks.NewMockClient()
		boltdb, _ := db.New(dbDir)

		projectLocker := &events.DefaultProjectLocker{
			Locker:    locking.NewClient(boltdb),
			VCSClient: vcsClientMock,
		}

		mockProjectCommandRunner := mocks.NewMockProjectCommandRunner()

		lockAcquiredRunner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               projectLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		thisCtx := models.ProjectCommandContext{
			Log:       logging.NewNoopLogger(t),
			Workspace: "default",
			Pull: models.PullRequest{
				BaseRepo: models.Repo{
					FullName: "locking-repo",
				},
				Num: 1,
			},
			RepoRelDir: ".",
		}

		When(mockProjectCommandRunner.PolicyCheck(
			matchers.EqModelsProjectCommandContext(thisCtx),
		)).ThenReturn(models.ProjectResult{
			Command:            models.PolicyCheckCommand,
			RepoRelDir:         thisCtx.RepoRelDir,
			Workspace:          thisCtx.Workspace,
			ProjectName:        thisCtx.ProjectName,
			PolicyCheckSuccess: &models.PolicyCheckSuccess{},
		})

		res := lockAcquiredRunner.PolicyCheck(thisCtx)

		Assert(t, res.PolicyCheckSuccess != nil, "policy check should be successful")
		mockProjectCommandRunner.VerifyWasCalledOnce().PolicyCheck(thisCtx)

		thatCtx := models.ProjectCommandContext{
			Log:       logging.NewNoopLogger(t),
			Workspace: "default",
			Pull: models.PullRequest{
				BaseRepo: models.Repo{
					FullName: "locking-repo",
				},
				Num: 2,
			},
			RepoRelDir: ".",
		}

		When(vcsClientMock.MarkdownPullLink(matchers.AnyModelsPullRequest())).ThenReturn("other PR", nil)

		mockProjectCommandRunner = mocks.NewMockProjectCommandRunner()
		lockFailed := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               projectLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     events.NewDefaultWorkingDirLocker(),
		}

		res = lockFailed.PolicyCheck(thatCtx)

		Assert(t, res.PolicyCheckSuccess == nil, "plan success should be nil")
		Equals(t, res.Failure, "This project is currently locked by an unapplied plan from pull other PR. To continue, delete the lock from other PR or apply that plan and merge the pull request.\n\nOnce the lock is released, comment `atlantis plan` here to re-plan.")
	})
}

// Test that it attempts to lock working dir when running apply command.
func TestLockingProjectCommandRunner_Apply(t *testing.T) {

	t.Run("working directory is locked", func(t *testing.T) {
		RegisterMockTestingT(t)
		mockLocker := eventsMocks.NewMockProjectLocker()
		mockProjectCommandRunner := mocks.NewMockProjectCommandRunner()
		mockWorkingDirLocker := eventsMocks.NewMockWorkingDirLocker()

		runner := runners.LockingProjectCommandRunner{
			ProjectCommandRunner: mockProjectCommandRunner,
			Locker:               mockLocker,
			LockURLGenerator:     mockURLGenerator{},
			WorkingDirLocker:     mockWorkingDirLocker,
		}

		ctx := models.ProjectCommandContext{
			Log:        logging.NewNoopLogger(t),
			Workspace:  "default",
			RepoRelDir: ".",
		}

		When(mockWorkingDirLocker.TryLock(
			AnyString(),
			AnyInt(),
			AnyString(),
		)).ThenReturn(nil, fmt.Errorf("workspace dir is locked"))

		res := runner.Apply(ctx)

		Equals(t, res.Error, fmt.Errorf("workspace dir is locked"))
		Assert(t, res.ApplySuccess == "", "apply should fail")
	})
}

type mockURLGenerator struct{}

func (m mockURLGenerator) GenerateLockURL(lockID string) string {
	return "https://" + lockID
}
