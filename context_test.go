package context

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing orchestra
type TestCtxSuite struct {
	suite.Suite
}

// Make sure that Account is set to five
// before each test
func (test *TestCtxSuite) SetupTest() {}

// Test_0_New tests the creation of the dev context.
func (test *TestCtxSuite) Test_0_New() {
	s := &test.Suite

	ctx, err := New("testdata/config.json")
	s.Require().NoError(err)
	s.Require().NotNil(ctx)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCtx(t *testing.T) {
	suite.Run(t, new(TestCtxSuite))
}
