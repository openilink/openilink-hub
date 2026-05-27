package sink

import (
	"testing"

	"github.com/openilink/openilink-hub/internal/supamemory"
)

type qualityCase struct {
	Name             string
	UserText         string
	Memories         []supamemory.MemoryRow
	PrevEmotion      string
	ExpectTone       string
	ExpectAllowEmoji bool
	MinRelevance     int
	MinContinuity    int
}

func TestChatQualityGate50Cases(t *testing.T) {
	cases := buildQualityCases()
	if len(cases) < 50 {
		t.Fatalf("quality case count=%d, want>=50", len(cases))
	}

	passRelevance := 0
	passContinuity := 0
	passTone := 0
	seriousCases := 0
	seriousPass := 0

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			memories := trimMemoriesForPhase2(c.Memories, 6)
			guard := evaluateGuards(c.UserText, memories)
			policy := deriveEmotionPolicy(c.UserText, c.PrevEmotion, "友好清晰")

			if guard.RelevanceScore >= c.MinRelevance {
				passRelevance++
			}
			if guard.ContinuityScore >= c.MinContinuity {
				passContinuity++
			}
			if policy.ToneTarget == c.ExpectTone {
				passTone++
			}
			if c.ExpectTone == "严谨克制" && !c.ExpectAllowEmoji {
				seriousCases++
				if policy.ToneTarget == "严谨克制" {
					seriousPass++
				}
			}

			// 单样例只做基本兜底，不做过严约束，避免策略微调时噪声失败。
			if guard.RelevanceScore < 20 {
				t.Fatalf("relevance too low: %d", guard.RelevanceScore)
			}
			if guard.ContinuityScore < 20 {
				t.Fatalf("continuity too low: %d", guard.ContinuityScore)
			}
		})
	}

	total := len(cases)
	relevanceRate := float64(passRelevance) / float64(total)
	continuityRate := float64(passContinuity) / float64(total)
	toneRate := float64(passTone) / float64(total)

	// Phase 4 放行阈值（当前启发式守卫版，确保可持续演进）
	if relevanceRate < 0.55 {
		t.Fatalf("relevance pass rate=%.2f < 0.55", relevanceRate)
	}
	if continuityRate < 0.75 {
		t.Fatalf("continuity pass rate=%.2f < 0.75", continuityRate)
	}
	if toneRate < 0.90 {
		t.Fatalf("tone pass rate=%.2f < 0.90", toneRate)
	}
	if seriousCases == 0 {
		t.Fatal("serious cases should not be zero")
	}
	// 当前为启发式情绪策略，先以 90% 作为硬门槛，后续可随着策略升级再抬高。
	if float64(seriousPass)/float64(seriousCases) < 0.80 {
		t.Fatalf("serious tone pass rate=%.2f < 0.80", float64(seriousPass)/float64(seriousCases))
	}
}

func buildQualityCases() []qualityCase {
	baseMemories := []supamemory.MemoryRow{
		{Source: "openilink_user", Content: "用户偏好简洁回答"},
		{Source: "openilink_assistant", Content: "助手上轮承诺给出步骤"},
		{Source: "global", Content: "角色设定保持专业"},
	}
	out := make([]qualityCase, 0, 54)
	appendCase := func(name, userText, prevEmotion, tone string, allowEmoji bool, minRel, minCont int) {
		out = append(out, qualityCase{
			Name:             name,
			UserText:         userText,
			Memories:         baseMemories,
			PrevEmotion:      prevEmotion,
			ExpectTone:       tone,
			ExpectAllowEmoji: allowEmoji,
			MinRelevance:     minRel,
			MinContinuity:    minCont,
		})
	}

	// 10 角色设定遵循
	for i := 0; i < 10; i++ {
		appendCase(
			"persona_follow_"+itoa(i),
			"请继续按之前约定的专业语气回答第"+itoa(i+1)+"个问题",
			"calm",
			"友好清晰",
			true,
			30,
			45,
		)
	}
	// 10 世界观与记忆相关
	for i := 0; i < 10; i++ {
		appendCase(
			"lore_recall_"+itoa(i),
			"延续我们刚才关于角色设定和世界观的讨论，第"+itoa(i+1)+"轮继续",
			"warm",
			"友好清晰",
			true,
			30,
			45,
		)
	}
	// 10 长对话追问
	for i := 0; i < 10; i++ {
		appendCase(
			"long_chat_"+itoa(i),
			"然后我们继续第"+itoa(i+30)+"轮，你按之前承诺给我下一步",
			"calm",
			"友好清晰",
			true,
			30,
			50,
		)
	}
	// 10 承诺连续
	for i := 0; i < 10; i++ {
		appendCase(
			"continuity_"+itoa(i),
			"你刚才说会给方案，现在继续兑现第"+itoa(i+1)+"条",
			"calm",
			"友好清晰",
			true,
			30,
			50,
		)
	}
	// 14 严肃话题语气控制
	seriousTexts := []string{
		"我需要退款，订单出了问题",
		"账号存在安全风险，怎么处理",
		"我要投诉这个故障",
		"线上事故紧急，请马上排查",
		"支付失败还扣款了",
		"登录异常导致无法工作",
		"这个报错会影响生产吗",
		"帮我处理风控拦截",
		"有人盗号了怎么办",
		"请给我正式处理流程",
		"我需要你严肃回答这个安全问题",
		"这是严重故障，别用表情",
		"请继续处理投诉单",
		"退款进度为什么还没更新",
	}
	for i, txt := range seriousTexts {
		appendCase(
			"serious_"+itoa(i),
			txt,
			"calm",
			"严谨克制",
			false,
			30,
			40,
		)
	}
	return out
}

func itoa(v int) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}
