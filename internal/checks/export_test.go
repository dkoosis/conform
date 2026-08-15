package checks

// Test hooks: the per-area checks stay unexported in the API; tests reach
// them directly so unit cases don't drag the whole runner (and its bd exec)
// along.
var (
	CheckMakefile     = checkMakefile
	CheckLintFloor    = checkLintFloor
	CheckLintPin      = checkLintPin
	CheckCIGate       = checkCIGate
	CheckCodexShape   = checkCodexShape
	CheckRetiredFiles = checkRetiredFiles
	CheckBDConfig     = checkBDConfig
	CheckHooksShape   = checkHooksShape
)

// Surface-2 test hooks.
var (
	CheckHooksPathFn = checkHooksPath
	CheckBDHooks     = checkBDHooks
	CheckReviewGate  = checkReviewGate
	CheckBDLive      = checkBDLive
)
