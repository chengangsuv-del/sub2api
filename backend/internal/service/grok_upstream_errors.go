package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// isGrokContentPolicyRejection identifies request-scoped safety refusals from
// xAI. These failures are caused by the prompt or media, so retrying another
// OAuth account cannot change the outcome and would incorrectly drain a pool.
// Keep this matcher deliberately narrow: account entitlement and suspension
// messages may mention policy but must retain the normal account failover path.
func isGrokContentPolicyRejection(statusCode int, responseBody []byte) bool {
	if statusCode != http.StatusForbidden || len(responseBody) == 0 {
		return false
	}
	if grokAccountAccessMessage(string(responseBody)) {
		return false
	}

	var payload any
	if json.Unmarshal(responseBody, &payload) == nil {
		if grokStructuredAccountAccessMarker(payload) {
			return false
		}
		if grokStructuredContentPolicyMarker(payload) {
			return true
		}
	}

	return grokContentPolicyMessage(string(responseBody))
}

// isGrokRequestScopedForbidden reports a 403 that names this request, not the
// account. xAI answers a prompt it will not run — including a jailbreak-style
// system prompt — with a bare 403 (permission-denied, an empty body, or a
// refusal that carries no suspension, subscription, credential, or quota
// evidence). Cooling or failing over treats that as an account outage and
// drains the pool. Account-evidence 403s stay on the normal cooldown path.
func isGrokRequestScopedForbidden(statusCode int, responseBody []byte) bool {
	if statusCode != http.StatusForbidden {
		return false
	}
	if isGrokContentPolicyRejection(statusCode, responseBody) {
		return true
	}
	if isGrokSpendingLimitError(responseBody) || grokForbiddenHasAccountEvidence(responseBody) {
		return false
	}
	return true
}

func grokForbiddenHasAccountEvidence(responseBody []byte) bool {
	if len(responseBody) == 0 {
		return false
	}
	if grokAccountAccessMessage(string(responseBody)) {
		return true
	}
	var payload any
	if json.Unmarshal(responseBody, &payload) == nil && grokStructuredAccountAccessMarker(payload) {
		return true
	}
	return false
}

func grokStructuredAccountAccessMarker(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			normalizedKey := normalizeGrokErrorMarker(key)
			switch normalizedKey {
			case "code", "error_code", "type", "category", "reason":
				if marker, ok := child.(string); ok && isGrokAccountAccessCode(marker) {
					return true
				}
			}
			if grokStructuredAccountAccessMarker(child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if grokStructuredAccountAccessMarker(child) {
				return true
			}
		}
	}
	return false
}

func grokStructuredContentPolicyMarker(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			normalizedKey := normalizeGrokErrorMarker(key)
			switch normalizedKey {
			case "code", "error_code", "type", "category", "reason":
				if marker, ok := child.(string); ok && isGrokContentPolicyCode(marker) {
					return true
				}
			}
			if grokStructuredContentPolicyMarker(child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if grokStructuredContentPolicyMarker(child) {
				return true
			}
		}
	}
	return false
}

func normalizeGrokErrorMarker(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isGrokContentPolicyCode(value string) bool {
	switch normalizeGrokErrorMarker(value) {
	case "content_filter",
		"content_policy",
		"content_policy_violation",
		"content_moderation",
		"cyber_policy",
		"new_sensitive":
		return true
	default:
		return false
	}
}

func isGrokAccountAccessCode(value string) bool {
	switch normalizeGrokErrorMarker(value) {
	case "account_suspended",
		"account_disabled",
		"user_suspended",
		"user_disabled",
		"subscription_required",
		"entitlement_required",
		"not_entitled",
		"plan_required":
		// permission-denied is omitted: xAI reuses it for both entitlement
		// refusals and request-scoped safety blocks, so the message decides.
		return true
	default:
		return false
	}
}

func grokAccountAccessMessage(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, phrase := range []string{
		"account suspended",
		"account has been suspended",
		"account disabled",
		"account has been disabled",
		"user suspended",
		"user has been suspended",
		"subscription required",
		"entitlement required",
		"not entitled",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func grokContentPolicyMessage(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}

	// xAI's media safety responses use these exact phrases. They are specific
	// enough not to classify a generic account-policy or entitlement message.
	for _, phrase := range []string{
		"the moderation feature is not available",
		"image is sensitive",
		"text is sensitive",
		"prohibited content",
		"forbidden content",
		"content policy violation",
		"content policy rejection",
		"content policy rejected",
		"content moderation rejection",
		"content moderation rejected",
		"content moderation blocked",
		"request blocked by content moderation",
		"request rejected by content moderation",
		"request blocked by policy",
		"request rejected by policy",
		"request violates policy",
		"prompt violates content policy",
		"prompt violates policy",
		"input violates content policy",
		"input violates policy",
		"violates usage guidelines",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	return false
}

const grokRequestScopedForbiddenClientMessage = "检测到违规 prefill，拒绝访问。请移除越狱或注入式 system prompt 后重试。密钥有效，这不是鉴权失败。"

// grokRequestScopedForbiddenClientMessage returns a stable client-facing
// refusal. DeepSeek Harness classifies any error text that starts with, or
// otherwise contains, 401/403 as AUTH and shows "API 密钥无效". The message
// must not contain those status codes, and the HTTP status written to the
// client must not be 401 or 403.
func grokRequestScopedForbiddenClientStatus() int {
	return http.StatusBadRequest
}

func (s *OpenAIGatewayService) writeGrokPromptRefusal(ctx context.Context, c *gin.Context, account *Account, resp *http.Response, body []byte) (*OpenAIForwardResult, error) {
	clientMsg := grokRequestScopedForbiddenClientMessage
	if s != nil {
		s.applyGrokForbiddenPolicy(ctx, account, body)
	}
	setOpsUpstreamError(c, resp.StatusCode, clientMsg, truncateString(string(body), 2048))
	if c != nil {
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, nil)
		c.Writer.Header().Del("Content-Length")
		MarkResponseCommitted(c)
		c.JSON(grokRequestScopedForbiddenClientStatus(), gin.H{
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": clientMsg,
			},
		})
	}
	return nil, fmt.Errorf("grok content policy rejection: %s", clientMsg)
}

// rewriteGrokRequestScopedForbiddenSSE replaces a streamed request-scoped
// refusal before it reaches the client. The original payload is retained for
// account classification; only the copy written downstream changes. A 401 or
// 403 anywhere in that copy makes DeepSeek Harness report an invalid API key.
func rewriteGrokRequestScopedForbiddenSSE(payload []byte) ([]byte, bool) {
	if !gjson.ValidBytes(payload) {
		return payload, false
	}
	updated := payload
	changed := false
	for _, path := range []string{"response.error.message", "error.message", "message", "error"} {
		value := gjson.GetBytes(updated, path)
		if value.Type != gjson.String {
			continue
		}
		next, err := sjson.SetBytes(updated, path, grokRequestScopedForbiddenClientMessage)
		if err != nil {
			return payload, false
		}
		updated = next
		changed = true
	}
	for _, path := range []string{"response.error.code", "error.code", "code"} {
		if gjson.GetBytes(updated, path).Type == gjson.Null {
			continue
		}
		next, err := sjson.SetBytes(updated, path, "invalid_request_error")
		if err != nil {
			return payload, false
		}
		updated = next
		changed = true
	}
	if !changed {
		return payload, false
	}
	return updated, true
}

// grokUpstreamErrorMessage extends the shared extractor with xAI's flat error
// body {"code":"...","error":"..."}. It is Grok-only on purpose: the shared
// extractUpstreamErrorMessage feeds failover, model-not-found and account
// disable decisions on every platform and must not drift with xAI's format.
// Non-string error values are never surfaced so unknown objects cannot leak.
func grokUpstreamErrorMessage(body []byte) string {
	if message := extractUpstreamErrorMessage(body); strings.TrimSpace(message) != "" {
		return message
	}
	if errValue := gjson.GetBytes(body, "error"); errValue.Type == gjson.String {
		return errValue.String()
	}
	return ""
}

// grokUpstreamErrorCode extends the shared extractor with xAI's top-level code.
// Nested error.code keeps priority. Like grokUpstreamErrorMessage it stays
// Grok-only: a top-level code would otherwise short-circuit the OpenAI-
// compatible model-not-found failover for bodies such as
// {"object":"error","message":"model not found","code":400}.
func grokUpstreamErrorCode(body []byte) string {
	if code := strings.TrimSpace(extractUpstreamErrorCode(body)); code != "" {
		return code
	}
	if code := gjson.GetBytes(body, "code"); code.Type == gjson.String || code.Type == gjson.Number {
		return strings.TrimSpace(code.String())
	}
	return ""
}

// shouldFailoverGrokUpstreamError is the body-aware counterpart of the
// status-only failover helper. Grok content refusals must stay on the current
// account and be returned to the caller instead of consuming the account pool.
// Free-usage / empty-output / billing bodies also failover even when the HTTP
// status alone would not (e.g. 400 with free-usage-exhausted).
func (s *OpenAIGatewayService) shouldFailoverGrokUpstreamError(statusCode int, responseBody []byte) bool {
	if isGrokRequestScopedForbidden(statusCode, responseBody) {
		return false
	}
	// A 422 emitted by xAI's ModelInput decoder is account/runtime compatibility,
	// not quota exhaustion. Another account may run a different upstream build,
	// so fail over without applying an account cooldown.
	if isGrokDecoderCompatibilityError(statusCode, responseBody) {
		return true
	}
	decision := classifyGrokUpstreamFailure(statusCode, responseBody, "")
	switch decision.Class {
	case GrokFailureFreeUsage, GrokFailureEmptyUpstream, GrokFailureBilling, GrokFailureModelCapacity, GrokFailureCompatibility:
		return decision.ShouldFailover
	}
	return s.shouldFailoverUpstreamError(statusCode)
}

func isGrokDecoderCompatibilityError(statusCode int, responseBody []byte) bool {
	if statusCode != http.StatusUnprocessableEntity || len(responseBody) == 0 {
		return false
	}
	for _, candidate := range grokStructuredErrorMessageCandidates(responseBody) {
		message := strings.ToLower(candidate)
		decoderSignal := strings.Contains(message, "untagged enum") ||
			strings.Contains(message, "decode") ||
			strings.Contains(message, "deserialize") ||
			strings.Contains(message, "deserializ") ||
			strings.Contains(message, "decoder")
		inputSignal := strings.Contains(message, "modelinput") ||
			strings.Contains(message, "model input") ||
			strings.Contains(message, "input[") ||
			strings.Contains(message, "input.")
		messageContentSignal := (strings.Contains(message, "messages[") ||
			strings.Contains(message, "messages.")) &&
			strings.Contains(message, "content") &&
			strings.Contains(message, "did not match any variant")
		if decoderSignal && (inputSignal || messageContentSignal) {
			return true
		}
	}
	return false
}

func grokStructuredErrorMessageCandidates(body []byte) []string {
	candidates := make([]string, 0, 6)
	appendCandidate := func(result gjson.Result) {
		if !result.Exists() {
			return
		}
		value := strings.TrimSpace(result.String())
		if value != "" {
			candidates = append(candidates, value)
		}
	}
	appendCandidate(gjson.GetBytes(body, "error.message"))
	appendCandidate(gjson.GetBytes(body, "error.error"))
	errorNode := gjson.GetBytes(body, "error")
	if errorNode.Type == gjson.String {
		appendCandidate(errorNode)
	}
	appendCandidate(gjson.GetBytes(body, "message"))
	appendCandidate(gjson.GetBytes(body, "detail"))
	if !json.Valid(body) {
		if plaintext := strings.TrimSpace(string(body)); plaintext != "" {
			candidates = append(candidates, plaintext)
		}
	}
	return candidates
}

// applyGrokForbiddenPolicy applies an administrator's existing temporary
// unschedulable rules to a non-content 403. It reports true only when a rule
// matched; unmatched responses retain the legacy entitlement cooldown.
func (s *OpenAIGatewayService) applyGrokForbiddenPolicy(ctx context.Context, account *Account, responseBody []byte) bool {
	if account == nil || !account.IsTempUnschedulableEnabled() {
		return false
	}

	matches := matchTempUnschedulableRules(account, http.StatusForbidden, responseBody)
	if len(matches) == 0 {
		return false
	}

	match := matches[0]
	// Reuse the central policy implementation when it has a repository. This
	// preserves the existing reason/cache format and avoids duplicating writes.
	if s != nil && s.rateLimitService != nil && s.rateLimitService.accountRepo != nil {
		stateCtx, cancel := openAIAccountStateContext(ctx)
		handled := s.rateLimitService.tryTempUnschedulable(
			stateCtx,
			account,
			http.StatusForbidden,
			responseBody,
		)
		cancel()
		if handled {
			return true
		}
	}

	// A partially constructed service (for example a unit-test gateway) still
	// honors the configured duration instead of silently falling back to 30m.
	cooldown := time.Duration(match.rule.DurationMinutes) * time.Minute
	if cooldown > 0 {
		s.tempUnscheduleGrok(ctx, account, cooldown, "grok configured forbidden rule")
	}
	return true
}
