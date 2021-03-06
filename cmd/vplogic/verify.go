/* SPDX-FileCopyrightText: © 2019-2020 Nadim Kobeissi <nadim@symbolic.software>
 * SPDX-License-Identifier: GPL-3.0-only */
// 458871bd68906e9965785ac87c2708ec

package vplogic

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Verify runs the main verification engine for Verifpal on a model loaded from a file.
// It returns a slice of verifyResults and a "results code".
func Verify(filePath string) ([]VerifyResult, string, error) {
	m, err := libpegParseModel(filePath, true)
	if err != nil {
		return []VerifyResult{}, "", err
	}
	return verifyModel(m)
}

func verifyModel(m Model) ([]VerifyResult, string, error) {
	valKnowledgeMap, valPrincipalStates, err := sanity(m)
	if err != nil {
		return []VerifyResult{}, "", err
	}
	initiated := time.Now().Format("03:04:05 PM")
	verifyAnalysisCountInit()
	verifyResultsInit(m)
	InfoMessage(fmt.Sprintf(
		"Verification initiated for '%s' at %s.", m.FileName, initiated,
	), "verifpal", false)
	switch m.Attacker {
	case "passive":
		err := verifyPassive(valKnowledgeMap, valPrincipalStates)
		if err != nil {
			return []VerifyResult{}, "", err
		}
	case "active":
		err := verifyActive(valKnowledgeMap, valPrincipalStates)
		if err != nil {
			return []VerifyResult{}, "", err
		}
	default:
		return []VerifyResult{}, "", fmt.Errorf("invalid attacker (%s)", m.Attacker)
	}
	fmt.Fprint(os.Stdout, "\n\n")
	return verifyEnd(m)
}

func verifyResolveQueries(
	valKnowledgeMap KnowledgeMap, valPrincipalState PrincipalState,
) {
	valVerifyResults, _ := verifyResultsGetRead()
	for _, verifyResult := range valVerifyResults {
		if !verifyResult.Resolved {
			queryStart(verifyResult.Query, valKnowledgeMap, valPrincipalState)
		}
	}
}

func verifyStandardRun(valKnowledgeMap KnowledgeMap, valPrincipalStates []PrincipalState, stage int) error {
	var scanGroup sync.WaitGroup
	var err error
	valAttackerState := attackerStateGetRead()
	for _, state := range valPrincipalStates {
		valPrincipalState := valueResolveAllPrincipalStateValues(state, valAttackerState)
		failedRewrites, _, valPrincipalState := valuePerformAllRewrites(valPrincipalState)
		err = sanityFailOnFailedCheckedPrimitiveRewrite(failedRewrites)
		if err != nil {
			return err
		}
		for i := range valPrincipalState.Assigned {
			err = sanityCheckEquationGenerators(valPrincipalState.Assigned[i], valPrincipalState)
			if err != nil {
				return err
			}
		}
		scanGroup.Add(1)
		verifyAnalysis(valKnowledgeMap, valPrincipalState, stage, &scanGroup)
		if err != nil {
			return err
		}
	}
	scanGroup.Wait()
	return err
}

func verifyPassive(valKnowledgeMap KnowledgeMap, valPrincipalStates []PrincipalState) error {
	InfoMessage("Attacker is configured as passive.", "info", false)
	phase := 0
	for phase <= valKnowledgeMap.MaxPhase {
		attackerStateInit(false)
		err := attackerStatePutPhaseUpdate(valPrincipalStates[0], phase)
		if err != nil {
			return err
		}
		err = verifyStandardRun(valKnowledgeMap, valPrincipalStates, 0)
		if err != nil {
			return err
		}
		phase = phase + 1
	}
	return nil
}

func verifyGetResultsCode(valVerifyResults []VerifyResult) string {
	resultsCode := ""
	for _, verifyResult := range valVerifyResults {
		q := ""
		r := ""
		switch verifyResult.Query.Kind {
		case "confidentiality":
			q = "c"
		case "authentication":
			q = "a"
		case "freshness":
			q = "f"
		case "unlinkability":
			q = "u"
		}
		switch verifyResult.Resolved {
		case true:
			r = "1"
		case false:
			r = "0"
		}
		resultsCode = fmt.Sprintf(
			"%s%s%s",
			resultsCode, q, r,
		)
	}
	return resultsCode
}

func verifyEnd(m Model) ([]VerifyResult, string, error) {
	var err error
	valVerifyResults, fileName := verifyResultsGetRead()
	for _, verifyResult := range valVerifyResults {
		if verifyResult.Resolved {
			InfoMessage(fmt.Sprintf(
				"%s: %s",
				prettyQuery(verifyResult.Query),
				verifyResult.Summary,
			), "result", false)
		}
	}
	completed := time.Now().Format("03:04:05 PM")
	InfoMessage(fmt.Sprintf(
		"Verification completed for '%s' at %s.", fileName, completed,
	), "verifpal", false)
	InfoMessage("Thank you for using Verifpal.", "verifpal", false)
	resultsCode := verifyGetResultsCode(valVerifyResults)
	if VerifHubScheduledShared {
		err = VerifHub(m, fileName, resultsCode)
	}
	return valVerifyResults, resultsCode, err
}
