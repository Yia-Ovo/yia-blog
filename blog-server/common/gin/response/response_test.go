package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/common"
	"github.com/gin-gonic/gin"
)

func TestWriteRespKeepsEnvelopeAndHidesInternalDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, errno := range []common.Errno{
		common.DatabaseErr.WithErr(errors.New("private-database-host SELECT secret")),
		common.ParamErr.WithMsg("private-binding-detail"),
		common.OK,
	} {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		WriteResp(ctx, map[string]string{"result": "kept"}, errno)
		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK || len(body) != 4 || body["code"] != float64(errno.Code) || body["msg"] != errno.Msg || body["err_msg"] != "" || body["data"].(map[string]interface{})["result"] != "kept" {
			t.Fatalf("response contract changed: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "private-") || strings.Contains(w.Body.String(), "SELECT secret") {
			t.Fatalf("internal details leaked: %s", w.Body.String())
		}
	}
}
