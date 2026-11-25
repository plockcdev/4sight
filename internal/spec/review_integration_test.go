package spec

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewAttemptsPreserveHistoryDolt(t *testing.T) {
	root := newMetadata(t)
	store := setupChangeImportDB(t, root)
	writeFile(t, filepath.Join(root, "coding.md"), "first change\n")
	var output, stderr bytes.Buffer
	if err := runChangeImport(root, store, &output, &stderr); err != nil {
		t.Fatal(err)
	}
	changeID, err := store.workspaceChangeID()
	if err != nil {
		t.Fatal(err)
	}
	var prompt bytes.Buffer
	if err := runChangeReviewPrompt(root, store, &prompt); err != nil {
		t.Fatalf("review prompt: %v", err)
	}
	for _, want := range []string{"one item at a time", "exact saved wording and source", "Do not infer approval from silence", "Do not edit specs or import changes", "Do not include rejected or deferred suggestions", "spec change author agent prompt", "Flag misplaced requirements", "location changes without SQL", "do not infer equivalence"} {
		if !strings.Contains(prompt.String(), want) {
			t.Errorf("review prompt missing %q", want)
		}
	}
	response := reviewResponse{Why: "why", Summary: "first review", Review: "first opinion", Ready: true, Files: []reviewFile{{Path: ".4sight/coding.md", Why: "why", Summary: "first file review", BodySummary: "first body summary"}}}
	if err := runChangeReviewResponse(root, store, changeID, response); err != nil {
		t.Fatalf("save first review: %v", err)
	}
	checkList := func(selector, want string) {
		t.Helper()
		var got bytes.Buffer
		if err := runChangeReviewList(store, selector, &got); err != nil {
			t.Fatalf("list %q: %v", selector, err)
		}
		if got.String() != want {
			t.Fatalf("list %q = %q, want %q", selector, got.String(), want)
		}
	}
	checkShow := func(selector string, required ...string) {
		t.Helper()
		var got bytes.Buffer
		if err := runChangeReviewShow(store, selector, &got); err != nil {
			t.Fatalf("show %q: %v", selector, err)
		}
		for _, value := range required {
			if !strings.Contains(got.String(), value) {
				t.Fatalf("show %q missing %q: %s", selector, value, got.String())
			}
		}
	}
	checkList("", changeID+": first review\n")
	checkList(changeID, "Attempt 1: first review (ready: 1)\n")
	checkShow("", "Review attempt: 1", "first opinion", "first body summary")
	checkShow(changeID+" 1", "Review attempt: 1", "first opinion", "first body summary")
	response.Summary = "second review"
	response.Review = "second opinion"
	response.Ready = false
	response.Files[0].Summary = "second file review"
	response.Files[0].BodySummary = "second body summary"
	if err := runChangeReviewResponse(root, store, changeID, response); err != nil {
		t.Fatalf("save second review: %v", err)
	}
	checkList("", changeID+": second review (2 attempts)\n")
	checkList(changeID, "Attempt 1: first review (ready: 1)\nAttempt 2: second review (ready: 0)\n")
	checkShow("", "Review attempt: 2", "second opinion", "second body summary")
	checkShow("1", "Review attempt: 1", "first opinion", "first body summary")
	checkShow(changeID+" 2", "Review attempt: 2", "second opinion", "second body summary")
	rows, err := store.queryCSV("SELECT cr.attempt,cr.summary,fr.summary,fr.body_summary FROM spec.change_review cr JOIN spec.file_change_reviews fr ON fr.change_review_id=cr.id WHERE cr.change_id=" + sqlString(changeID) + " ORDER BY cr.attempt")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || strings.Join(rows[0], "|") != "1|first review|first file review|first body summary" || strings.Join(rows[1], "|") != "2|second review|second file review|second body summary" {
		t.Fatalf("review history: %v", rows)
	}
	rows, err = store.queryCSV("SELECT summary FROM spec.bodies WHERE state='workspace'")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != "second body summary" {
		t.Fatalf("workspace summary: %v", rows)
	}
	for _, tc := range []struct{ selector, want, absent string }{{changeID, "second opinion", "first opinion"}, {changeID + " 1", "first opinion", "second opinion"}, {changeID + " 2", "second opinion", "first opinion"}} {
		var report bytes.Buffer
		if err := runChangeReviewShow(store, tc.selector, &report); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(report.String(), tc.want) || strings.Contains(report.String(), tc.absent) {
			t.Fatalf("show %s: %s", tc.selector, report.String())
		}
	}
}
