package spec

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed review_discussion.txt
var reviewDiscussion string

type reviewState struct{ Workspace, Submitted, Prompt string }
type reviewFile struct {
	Path        string `json:"path"`
	Why         string `json:"why"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Details     string `json:"details"`
	BodySummary string `json:"body_summary"`
}
type reviewResponse struct {
	Why         string       `json:"why"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Details     string       `json:"details"`
	Review      string       `json:"review"`
	Ready       bool         `json:"ready"`
	Files       []reviewFile `json:"file_reviews"`
}

func ChangeReviewAgentPrompt(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeReviewPrompt(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func ChangeReviewShow(cwd, changeID string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeReviewShow(newDoltStore(filepath.Join(root, "db")), changeID, os.Stdout)
}

func ChangeReviewList(cwd, changeID string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeReviewList(newDoltStore(filepath.Join(root, "db")), changeID, os.Stdout)
}

func runChangeReviewList(st *doltStore, changeID string, out io.Writer) error {
	if changeID != "" {
		exists, err := st.queryCSV("SELECT id FROM spec.changes WHERE id=" + sqlString(changeID))
		if err != nil {
			return err
		}
		if len(exists) == 0 {
			return fmt.Errorf("change %s does not exist", changeID)
		}
		rows, err := st.queryCSV("SELECT attempt,summary,ready FROM spec.change_review WHERE change_id=" + sqlString(changeID) + " ORDER BY attempt")
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return fmt.Errorf("no review is available for change %s", changeID)
		}
		for _, r := range rows {
			if _, err := fmt.Fprintf(out, "Attempt %s: %s (ready: %s)\n", r[0], r[1], r[2]); err != nil {
				return err
			}
		}
		return nil
	}
	rows, err := st.queryCSV("SELECT c.id,r.summary,(SELECT COUNT(*) FROM spec.change_review x WHERE x.change_id=c.id) FROM spec.changes c JOIN spec.change_review r ON r.change_id=c.id AND r.attempt=(SELECT MAX(x.attempt) FROM spec.change_review x WHERE x.change_id=c.id) ORDER BY c.created_at,c.id")
	if err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(out, "%s: %s", r[0], r[1]); err != nil {
			return err
		}
		if r[2] != "1" {
			if _, err := fmt.Fprintf(out, " (%s attempts)", r[2]); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	return nil
}

type changeReviewFileReport struct {
	Path, Why, Summary, Description, Details, BodySummary string
}

func runChangeReviewShow(st *doltStore, changeID string, out io.Writer) error {
	parts := strings.Fields(changeID)
	attempt := ""
	changeID = ""
	if len(parts) > 0 {
		if _, err := strconv.Atoi(parts[0]); err == nil {
			attempt = parts[0]
		} else {
			changeID = parts[0]
		}
	}
	if len(parts) > 1 {
		attempt = parts[1]
	}
	if attempt != "" {
		n, err := strconv.Atoi(attempt)
		if err != nil || n < 1 {
			return fmt.Errorf("attempt must be a positive integer")
		}
	}
	if changeID == "" {
		state, err := st.reviewState()
		if err != nil {
			return err
		}
		changeID = state.Workspace
	}
	exists, err := st.queryCSV("SELECT id FROM spec.changes WHERE id=" + sqlString(changeID))
	if err != nil {
		return fmt.Errorf("look up change %s: %w", changeID, err)
	}
	if len(exists) == 0 {
		return fmt.Errorf("change %s does not exist", changeID)
	}
	query := "SELECT why,summary,description,details,review,ready,attempt,id FROM spec.change_review WHERE change_id=" + sqlString(changeID)
	if attempt != "" {
		query += " AND attempt=" + attempt
	}
	query += " ORDER BY attempt DESC LIMIT 1"
	reviews, err := st.queryCSV(query)
	if err != nil {
		return fmt.Errorf("look up review for change %s: %w", changeID, err)
	}
	if len(reviews) == 0 {
		if attempt != "" {
			return fmt.Errorf("no such review attempt %s is available for change %s", attempt, changeID)
		}
		return fmt.Errorf("no review is available for change %s", changeID)
	}
	if len(reviews[0]) < 6 {
		return fmt.Errorf("incomplete review data for change %s", changeID)
	}
	rows, err := st.queryCSV("SELECT d.path,p.name,r.why,r.summary,r.description,r.details,r.body_summary FROM spec.file_change_reviews r JOIN spec.change_review cr ON cr.id=r.change_review_id JOIN spec.file_changes fc ON fc.id=r.file_change_id JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id WHERE cr.id=" + reviews[0][7])
	if err != nil {
		return fmt.Errorf("look up file reviews for change %s: %w", changeID, err)
	}
	files := make([]changeReviewFileReport, 0, len(rows))
	for _, row := range rows {
		if len(row) < 7 {
			return fmt.Errorf("incomplete file review data for change %s", changeID)
		}
		path := row[0]
		if path != "" {
			path += "/"
		}
		path = ".4sight/" + path + row[1]
		files = append(files, changeReviewFileReport{
			Path: path, Why: row[2], Summary: row[3], Description: row[4], Details: row[5], BodySummary: row[6],
		})
	}
	slices.SortFunc(files, func(a, b changeReviewFileReport) int { return strings.Compare(a.Path, b.Path) })
	ready := reviews[0][5] == "1" || strings.EqualFold(reviews[0][5], "true")
	if _, err := fmt.Fprintf(out, "Change ID: %s\nReview attempt: %s\n\nOverall review:\n", changeID, reviews[0][6]); err != nil {
		return err
	}
	overallFields := [][2]string{
		{"Why", reviews[0][0]}, {"Summary", reviews[0][1]}, {"Description", reviews[0][2]},
		{"Details", reviews[0][3]}, {"Review", reviews[0][4]}, {"Ready", fmt.Sprint(ready)},
	}
	for _, field := range overallFields {
		if err := writeReviewField(out, "  ", field[0], field[1]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "\nFile reviews:"); err != nil {
		return err
	}
	if len(files) == 0 {
		_, err := fmt.Fprintln(out, "  (none)")
		return err
	}
	for i, file := range files {
		if i > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		fields := [][2]string{
			{"Path", file.Path}, {"Why", file.Why}, {"Summary", file.Summary},
			{"Description", file.Description}, {"Details", file.Details}, {"Body summary", file.BodySummary},
		}
		for _, field := range fields {
			if err := writeReviewField(out, "  ", field[0], field[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeReviewField(out io.Writer, indent, label, value string) error {
	lines := strings.Split(value, "\n")
	if value == "" {
		_, err := fmt.Fprintf(out, "%s%s: \n", indent, label)
		return err
	}
	if _, err := fmt.Fprintf(out, "%s%s: %s\n", indent, label, lines[0]); err != nil {
		return err
	}
	for _, line := range lines[1:] {
		if _, err := fmt.Fprintf(out, "%s  %s\n", indent, line); err != nil {
			return err
		}
	}
	return nil
}

func runChangeReviewPrompt(root string, st *doltStore, out io.Writer) error {
	state, err := st.reviewState()
	if err != nil {
		return fmt.Errorf("read current changes: %w", err)
	}
	meta := fmt.Sprintf("Change being reviewed (workspace change ID): %s\nWorkspace change: %s\nSubmitted change: %s\nChange prompt: %s\n", state.Workspace, state.Workspace, state.Submitted, nonempty(state.Prompt, "(empty)"))
	if state.Workspace == state.Submitted {
		_, err = fmt.Fprintln(out, meta+"There is no unsubmitted workspace change. Submit a request prompt or import spec changes, then run this command again.")
		return err
	}
	bad, err := st.workspaceSyncProblems(root)
	if err != nil {
		return err
	}
	if len(bad) > 0 {
		_, err = fmt.Fprintf(out, "%sWorkspace specs are out of sync:\n%s\nImport spec changes or overwrite the affected local files from the database, then retry.\n", meta, strings.Join(bad, "\n"))
		return err
	}
	if state.Prompt != "" {
		_, err = fmt.Fprintf(out, "%sRun `4sight spec change author agent prompt` to author the requested spec changes, then import them before reviewing.\n", meta)
		return err
	}
	reviews, err := st.queryCSV("SELECT COALESCE(MAX(attempt),0) FROM spec.change_review WHERE change_id=" + sqlString(state.Workspace))
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(reviews[0][0])
	if err != nil {
		return err
	}
	meta += fmt.Sprintf("Next review attempt: %d\nIf a review already exists, proceed only for an explicitly requested re-review; otherwise ask the user to request a re-review. Include the user’s new context and preserve earlier attempts.\n", n+1)
	unreviewedPaths, err := st.unreviewedPaths(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	if state.Prompt == "" && len(unreviewedPaths) == 0 && n == 0 {
		_, err = fmt.Fprintln(out, meta+"There is nothing to review.")
		return err
	}
	paths, err := st.changedPaths(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	for i := range paths {
		paths[i] = ".4sight/" + paths[i]
	}
	relationships, err := st.relationshipDetails(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "changes", state.Workspace), 0755); err != nil {
		return err
	}
	diff, err := st.compositeDiff(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	diffPath := filepath.Join(root, "changes", state.Workspace, "unified.diff")
	if err := os.WriteFile(diffPath, []byte(diff), 0644); err != nil {
		return err
	}
	bodies, err := st.listWorkspaceBodies()
	if err != nil {
		return err
	}
	var workspaceDBSpecs, workspaceCodeSpecs []string
	for _, d := range bodies {
		if d.Deleted || isTextAssetDir(d.Dir) {
			continue
		}
		path := specRelPath(d.Dir, d.Name)
		if d.Dir == "schemas" {
			workspaceDBSpecs = append(workspaceDBSpecs, path)
		} else {
			workspaceCodeSpecs = append(workspaceCodeSpecs, path)
		}
	}
	slices.Sort(workspaceDBSpecs)
	slices.Sort(workspaceCodeSpecs)
	removed, err := st.removedSpecPaths()
	if err != nil {
		return err
	}
	var removedDBSpecs, removedCodeSpecs []string
	for _, p := range removed {
		if p.dir == "schemas" {
			removedDBSpecs = append(removedDBSpecs, specRelPath(p.dir, p.name))
		} else {
			removedCodeSpecs = append(removedCodeSpecs, specRelPath(p.dir, p.name))
		}
	}
	schemas, err := listSchemaSpecs(filepath.Join(root, "schemas"))
	if err != nil {
		return err
	}
	projectRoot := filepath.Dir(root)
	var b strings.Builder
	b.WriteString("Review the current unsubmitted specification changes for quality and consistency.\n\n")
	b.WriteString("Check that schema specs define persisted data and constraints, while application design and command specs specify operational choices such as database location, initialization, and access. Flag misplaced requirements and missing cross-spec behavior, including application work needed for location changes without SQL. For changed schemas, assess whether migration requirements and any no-SQL conclusion are clear enough to implement safely; do not infer equivalence from descriptive wording or checksums alone.\n\n")
	b.WriteString("Dolt is the default when no different database is established, using file-backed access without a daemon; other database backends have not been tested. Users should commit Dolt database files alongside application changes; Git-based database rollback is deferred. Respect project-specific requirements. Treat 4sight's working database bootstrap and migration workflow as established. For database-backed applications, assess initialization and upgrade against the following default, respecting an established alternative or an explicit exclusion of migration support. Do not assume 4sight's working-database migrations upgrade deployed instances. Identify missing application access, deployment lifecycle, or existing-data import decisions when they matter to the specified behavior. Check that newly introduced managed databases can use the standard application migration tracking requirements.\n\n")
	b.WriteString(applicationDatabaseLifecycle)
	b.WriteString("\n")
	b.WriteString("\nCurrent Workspace DB Specs\n")
	for _, p := range workspaceDBSpecs {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	b.WriteString("\nCurrent Workspace Specs\n")
	for _, p := range workspaceCodeSpecs {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	b.WriteString("\nRemoved DB Specs\n")
	if len(removedDBSpecs) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, p := range removedDBSpecs {
			fmt.Fprintf(&b, "- %s\n", p)
		}
	}
	b.WriteString("\nRemoved Code Specs\n")
	if len(removedCodeSpecs) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, p := range removedCodeSpecs {
			fmt.Fprintf(&b, "- %s\n", p)
		}
	}
	b.WriteString("\nDatabase Schema Specs\n")
	for _, schema := range schemas {
		fmt.Fprintf(&b, "- %s\n", relFromProject(projectRoot, schema))
	}
	diffProjectPath := filepath.ToSlash(filepath.Join(".4sight", "changes", state.Workspace, "unified.diff"))
	fmt.Fprintf(&b, "\nComposite diff: %s\n\nRead the composite diff to review all changes since the last submitted specification. Read the full workspace spec for every file referenced in the diff, read all database schema specs listed above, and read workspace specs related to changed files or the change prompt, including parent command design docs and coding standards as applicable. Follow applicable coding standards, including referenced standards and command-path overrides, and assess changed specifications against them, unless directly instructed otherwise by the user.\n", diffProjectPath)
	b.WriteString("Call out a mismatch between an edited design and a parent-scoped design, even if the mismatch was not introduced by the change. When a higher-scope design is edited, assess whether design documents beneath it remain aligned.\n")
	responseSchemaRel := filepath.Join("subcommands", "spec", "subcommands", "change", "subcommands", "review", "subcommands", "agent", "responseSchema.json")
	responseSchemaPath := filepath.Join(root, responseSchemaRel)
	if _, err := os.Stat(responseSchemaPath); err == nil {
		responseSchema := filepath.ToSlash(filepath.Join(".4sight", responseSchemaRel))
		fmt.Fprintf(&b, "\nRead the response schema at %s and follow its constraints.\n", responseSchema)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check response schema %s: %w", responseSchemaPath, err)
	}
	b.WriteString("\nChanged spec files to review:\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	if len(relationships) > 0 {
		b.WriteString("\nInferred file relationships (review as suggestions, not confirmed identity):\n")
		for _, relationship := range relationships {
			fmt.Fprintf(&b, "- %s\n", relationship)
		}
	}
	fmt.Fprintf(&b, "\nSubmit a qualitative review of every specification change after the submitted pointer through the workspace pointer, including earlier unsubmitted changes. The overall why, summary, description, details, and review must cover this full range, not just the latest change. Include one file_reviews entry for every file changed in this range. Send one JSON object to `4sight spec change review agent response %s` on stdin. This argument must be the change being reviewed ID shown above. Each file entry needs path, why, summary (max 65 characters), description, details, and body_summary (max 65 characters). Include overall why, summary (max 65 characters), description, details, review, and ready boolean. JSON shape:\n", state.Workspace)
	b.WriteString("```json\n{\"why\":\"...\",\"summary\":\"...\",\"description\":\"...\",\"details\":\"...\",\"review\":\"...\",\"ready\":true,\"file_reviews\":[{\"path\":\".4sight/subcommands/example/cmd.md\",\"why\":\"...\",\"summary\":\"...\",\"description\":\"...\",\"details\":\"...\",\"body_summary\":\"...\"}]}\n```\n")
	b.WriteString(reviewDiscussion)
	_, err = io.WriteString(out, b.String())
	return err
}

func nonempty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func ChangeReviewAgentResponse(cwd, changeID string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	state, err := st.reviewState()
	if err != nil {
		return err
	}
	if changeID != state.Workspace {
		return fmt.Errorf("change ID %s is not the latest workspace change %s; submit reviews only for the current workspace change", changeID, state.Workspace)
	}
	data, err := readReviewStdin(os.Stdin, time.Second)
	if err != nil {
		return err
	}
	if err := validateReviewJSONShape(data); err != nil {
		return fmt.Errorf("invalid review JSON: %w", err)
	}
	var response reviewResponse
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&response); err != nil {
		return fmt.Errorf("invalid review JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("review input must contain exactly one JSON object")
	}
	if err := runChangeReviewResponse(root, st, changeID, response); err != nil {
		return err
	}
	rows, err := st.queryCSV("SELECT MAX(attempt) FROM spec.change_review WHERE change_id=" + sqlString(changeID))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "Saved review for change %s, attempt %s.\n", changeID, rows[0][0])
	return err
}

func readReviewStdin(r io.Reader, timeout time.Duration) ([]byte, error) {
	ch := make(chan struct {
		b []byte
		e error
	}, 1)
	go func() {
		b, e := io.ReadAll(r)
		ch <- struct {
			b []byte
			e error
		}{b, e}
	}()
	select {
	case v := <-ch:
		if v.e != nil {
			return nil, v.e
		}
		return v.b, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for review data on stdin; pipe the required JSON review to this command")
	}
}

func validateReviewJSONShape(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("review must be a JSON object")
	}
	for _, name := range []string{"why", "summary", "description", "details", "review", "ready", "file_reviews"} {
		if _, ok := fields[name]; !ok {
			return fmt.Errorf("missing required field %q", name)
		}
	}
	for _, name := range []string{"why", "summary", "description", "details", "review"} {
		if !hasJSONType(fields[name], '"') {
			return fmt.Errorf("field %q must be a string", name)
		}
	}
	ready := bytes.TrimSpace(fields["ready"])
	if string(ready) != "true" && string(ready) != "false" {
		return fmt.Errorf("field %q must be a boolean", "ready")
	}
	if !hasJSONType(fields["file_reviews"], '[') {
		return fmt.Errorf("field %q must be an array", "file_reviews")
	}
	var fileReviews []map[string]json.RawMessage
	if err := json.Unmarshal(fields["file_reviews"], &fileReviews); err != nil {
		return fmt.Errorf("field %q must be an array of objects: %w", "file_reviews", err)
	}
	for i, fileReview := range fileReviews {
		if fileReview == nil {
			return fmt.Errorf("file_reviews[%d] must be an object", i)
		}
		for _, name := range []string{"path", "why", "summary", "description", "details", "body_summary"} {
			value, ok := fileReview[name]
			if !ok {
				return fmt.Errorf("file_reviews[%d] is missing required field %q", i, name)
			}
			if !hasJSONType(value, '"') {
				return fmt.Errorf("file_reviews[%d].%s must be a string", i, name)
			}
		}
	}
	return nil
}

func hasJSONType(value json.RawMessage, firstByte byte) bool {
	value = bytes.TrimSpace(value)
	return len(value) > 0 && value[0] == firstByte
}

func validateReview(r reviewResponse) error {
	if strings.TrimSpace(r.Why) == "" || strings.TrimSpace(r.Summary) == "" || len([]rune(r.Summary)) > 65 || strings.TrimSpace(r.Review) == "" {
		return fmt.Errorf("overall why, review, and summary (1-65 characters) are required")
	}
	if len(r.Files) == 0 {
		return fmt.Errorf("at least one file review is required")
	}
	seen := map[string]bool{}
	for _, f := range r.Files {
		if f.Path == "" || seen[f.Path] {
			return fmt.Errorf("file review paths must be non-empty and unique")
		}
		seen[f.Path] = true
		if strings.TrimSpace(f.Why) == "" || strings.TrimSpace(f.Summary) == "" || len([]rune(f.Summary)) > 65 || strings.TrimSpace(f.BodySummary) == "" || len([]rune(f.BodySummary)) > 65 {
			return fmt.Errorf("file review %q requires why, summary (1-65 characters), and body_summary", f.Path)
		}
	}
	return nil
}

func runChangeReviewResponse(root string, st *doltStore, changeID string, r reviewResponse) error {
	if err := validateReview(r); err != nil {
		return err
	}
	state, err := st.reviewState()
	if err != nil {
		return err
	}
	if state.Workspace != changeID {
		return fmt.Errorf("change ID %s is not the latest workspace change %s; submit reviews only for the current workspace change", changeID, state.Workspace)
	}
	bad, err := st.workspaceSyncProblems(root)
	if err != nil {
		return err
	}
	if len(bad) > 0 {
		return fmt.Errorf("filesystem and workspace bodies are out of sync; run `4sight spec change import` before reviewing: %s", strings.Join(bad, "; "))
	}
	expected, err := st.changedPaths(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	wantPaths := make(map[string]bool, len(expected))
	for _, p := range expected {
		wantPaths[p] = true
	}
	for _, f := range r.Files {
		p := strings.TrimPrefix(f.Path, ".4sight/")
		if !wantPaths[p] {
			return fmt.Errorf("file review path %q is not a changed file between submitted and workspace", f.Path)
		}
		delete(wantPaths, p)
	}
	if len(wantPaths) != 0 {
		var missing []string
		for p := range wantPaths {
			missing = append(missing, p)
		}
		slices.Sort(missing)
		return fmt.Errorf("missing file reviews for: %s", strings.Join(missing, ", "))
	}
	resolved := make([]resolvedReviewFile, 0, len(r.Files))
	for _, f := range r.Files {
		file, err := st.resolveReviewFile(state, f)
		if err != nil {
			return err
		}
		resolved = append(resolved, file)
	}
	ready := 0
	if r.Ready {
		ready = 1
	}
	changeName, err := st.nextReviewName()
	if err != nil {
		return err
	}
	var tx strings.Builder
	tx.WriteString("START TRANSACTION;\n")
	fmt.Fprintf(&tx, "INSERT INTO spec.change_review (name,change_id,attempt,why,summary,description,details,ready,review) SELECT %s,%s,(SELECT COALESCE(MAX(attempt),0)+1 FROM spec.change_review WHERE change_id=%s),%s,%s,%s,%s,%d,%s FROM spec.current_changes WHERE singleton=1 AND workspace_change_id=%s;\n", sqlString(changeName), sqlString(changeID), sqlString(changeID), sqlBodyLiteral(r.Why), sqlBodyLiteral(r.Summary), sqlBodyLiteral(r.Description), sqlBodyLiteral(r.Details), ready, sqlBodyLiteral(r.Review), sqlString(changeID))
	for _, file := range resolved {
		fmt.Fprintf(&tx, "INSERT INTO spec.file_change_reviews (file_change_id,change_review_id,why,summary,body_summary,description,details) VALUES (%s,(SELECT id FROM spec.change_review WHERE name=%s),%s,%s,%s,%s,%s);\n", file.FileChangeID, sqlString(changeName), sqlBodyLiteral(file.Review.Why), sqlBodyLiteral(file.Review.Summary), sqlBodyLiteral(file.Review.BodySummary), sqlBodyLiteral(file.Review.Description), sqlBodyLiteral(file.Review.Details))
		fmt.Fprintf(&tx, "UPDATE spec.bodies SET summary=%s WHERE file_id=%s AND state='workspace';\n", sqlBodyLiteral(file.Review.BodySummary), file.FileID)
	}
	tx.WriteString("COMMIT;")
	if err := st.run("sql", "-q", tx.String()); err != nil {
		return fmt.Errorf("save change review: %w", err)
	}
	return nil
}

func (s *doltStore) reviewState() (reviewState, error) {
	rows, err := s.queryCSV("SELECT c.workspace_change_id,c.submitted_change_id,ch.prompt FROM spec.current_changes c JOIN spec.changes ch ON ch.id=c.workspace_change_id WHERE c.singleton=1")
	if err != nil {
		return reviewState{}, err
	}
	if len(rows) == 0 || len(rows[0]) < 3 {
		return reviewState{}, fmt.Errorf("current changes are not initialized")
	}
	return reviewState{rows[0][0], rows[0][1], rows[0][2]}, nil
}

func (s *doltStore) workspaceSyncProblems(root string) ([]string, error) {
	docs, err := s.listWorkspaceBodies()
	if err != nil {
		return nil, err
	}
	disk, err := listOnDiskCodeSpecs(root)
	if err != nil {
		return nil, err
	}
	db := map[specKey]string{}
	for _, d := range docs {
		if !d.Deleted {
			db[specKey{d.Dir, d.Name}] = d.Body
		}
	}
	keys := map[specKey]bool{}
	for k := range disk {
		keys[k] = true
	}
	for k := range db {
		keys[k] = true
	}
	var out []string
	for k := range keys {
		body, onDisk, e := diskSpec(root, disk, k)
		if e != nil {
			return nil, e
		}
		diskText := ""
		if onDisk {
			diskText = normalizeTrackedBody(k.dir, k.name, body)
		}
		dbText := normalizeTrackedBody(k.dir, k.name, db[k])
		_, inDB := db[k]
		if inDB == onDisk && (!inDB || dbText == diskText) {
			continue
		}
		plus, minus := lineChangeCounts(dbText, diskText)
		out = append(out, fmt.Sprintf("%s is out of sync (+%d/-%d on disk vs workspace body)", specRelPath(k.dir, k.name), plus, minus))
	}
	slices.Sort(out)
	return out, nil
}

func (s *doltStore) unreviewedPaths(submitted, workspace string) ([]string, error) {
	rows, err := s.queryCSV("SELECT d.path,p.name FROM spec.file_changes fc JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(workspace) + " AND w.parent_id=fc.change_id LEFT JOIN spec.file_change_reviews r ON r.file_change_id=fc.id WHERE a.depth>0 AND r.id IS NULL AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) GROUP BY d.path,p.name ORDER BY d.path,p.name")
	if err != nil {
		return nil, err
	}
	var p []string
	for _, r := range rows {
		p = append(p, specRelPath(r[0], r[1]))
	}
	return p, nil
}

func (s *doltStore) changedPaths(submitted, workspace string) ([]string, error) {
	rows, err := s.queryCSV("SELECT d.path,p.name FROM spec.file_changes fc JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(workspace) + " AND w.parent_id=fc.change_id WHERE a.depth>0 AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) GROUP BY d.path,p.name ORDER BY d.path,p.name")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, row := range rows {
		if len(row) >= 2 {
			paths = append(paths, specRelPath(row[0], row[1]))
		}
	}
	return paths, nil
}

func (s *doltStore) relationshipDetails(submitted, workspace string) ([]string, error) {
	query := "SELECT d.path,p.name,COALESCE(sd.path,''),COALESCE(sp.name,''),fc.related_file_id,fc.similarity FROM spec.file_changes fc JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(workspace) + " AND w.parent_id=fc.change_id LEFT JOIN spec.files sf ON sf.id=fc.related_file_id LEFT JOIN spec.filepaths sp ON sp.id=sf.current_filepath_id LEFT JOIN spec.dirs sd ON sd.id=sp.dir_id WHERE a.depth>0 AND fc.similarity IS NOT NULL AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) ORDER BY d.path,p.name"
	rows, err := s.queryCSV(query)
	if err != nil {
		return nil, err
	}
	var details []string
	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		path := specRelPath(row[0], row[1])
		if row[4] == "" {
			details = append(details, fmt.Sprintf("possible rename .4sight/%s (similarity %s)", path, row[5]))
		} else {
			source := specRelPath(row[2], row[3])
			details = append(details, fmt.Sprintf("possible copy .4sight/%s from .4sight/%s (similarity %s)", path, source, row[5]))
		}
	}
	return details, nil
}

func (s *doltStore) removedSpecPaths() ([]specKey, error) {
	rows, err := s.queryCSV("SELECT ds.path,ps.name,COALESCE(bs.deleted,0),COALESCE(bw.deleted,0) FROM spec.files f LEFT JOIN spec.bodies bs ON bs.file_id=f.id AND bs.state='submitted' LEFT JOIN spec.file_changes fcs ON fcs.id=bs.file_change_id LEFT JOIN spec.filepaths ps ON ps.id=fcs.filepath_id LEFT JOIN spec.dirs ds ON ds.id=ps.dir_id LEFT JOIN spec.bodies bw ON bw.file_id=f.id AND bw.state='workspace' ORDER BY ds.path,ps.name")
	if err != nil {
		return nil, err
	}
	paths := make([]specKey, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 || row[2] == "1" || row[3] != "1" || row[0] == "" || row[1] == "" || isTextAssetDir(row[0]) {
			continue
		}
		paths = append(paths, specKey{dir: row[0], name: row[1]})
	}
	return paths, nil
}

func (s *doltStore) compositeDiff(submitted, workspace string) (string, error) {
	fromState, err := s.bodyStateForChange(submitted)
	if err != nil {
		return "", err
	}
	toState, err := s.bodyStateForChange(workspace)
	if err != nil {
		return "", err
	}
	rows, err := s.queryCSV("SELECT COALESCE(ds.path,''),COALESCE(ps.name,''),COALESCE(dw.path,''),COALESCE(pw.name,''),COALESCE(bs.body,''),COALESCE(bw.body,''),COALESCE(bs.deleted,0),COALESCE(bw.deleted,0),bs.file_id,bw.file_id FROM spec.files f LEFT JOIN spec.bodies bs ON bs.file_id=f.id AND bs.state=" + sqlString(fromState) + " LEFT JOIN spec.file_changes fcs ON fcs.id=bs.file_change_id LEFT JOIN spec.filepaths ps ON ps.id=fcs.filepath_id LEFT JOIN spec.dirs ds ON ds.id=ps.dir_id LEFT JOIN spec.bodies bw ON bw.file_id=f.id AND bw.state=" + sqlString(toState) + " LEFT JOIN spec.file_changes fcw ON fcw.id=bw.file_change_id LEFT JOIN spec.filepaths pw ON pw.id=fcw.filepath_id LEFT JOIN spec.dirs dw ON dw.id=pw.dir_id WHERE bs.file_id IS NOT NULL OR bw.file_id IS NOT NULL ORDER BY ds.path,ps.name,dw.path,pw.name")
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, r := range rows {
		if len(r) < 10 {
			continue
		}
		oldPath, newPath := specRelPath(r[0], r[1]), specRelPath(r[2], r[3])
		old, new := r[4], r[5]
		if r[6] == "1" {
			old = ""
		}
		if r[7] == "1" {
			new = ""
		}
		if old == new && oldPath == newPath && r[6] == r[7] {
			continue
		}
		if oldPath == "" || oldPath == "." {
			oldPath = newPath
		}
		if newPath == "" || newPath == "." {
			newPath = oldPath
		}
		oldDiffPath, newDiffPath := ".4sight/"+oldPath, ".4sight/"+newPath
		if r[6] == "1" {
			oldDiffPath = "/dev/null"
		}
		if r[7] == "1" {
			newDiffPath = "/dev/null"
		}
		if r[8] == "" {
			oldDiffPath = "/dev/null"
		}
		if r[9] == "" {
			newDiffPath = "/dev/null"
		}
		diff := unifiedDiffPaths(oldDiffPath, newDiffPath, old, new)
		if diff != "" {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(diff)
		}
	}
	return out.String(), nil
}

func (s *doltStore) bodyStateForChange(changeID string) (string, error) {
	rows, err := s.queryCSV("SELECT implemented_change_id,submitted_change_id,workspace_change_id FROM spec.current_changes WHERE singleton=1")
	if err != nil {
		return "", err
	}
	if len(rows) != 1 || len(rows[0]) < 3 {
		return "", fmt.Errorf("current change pointers are not initialized")
	}
	if changeID == rows[0][2] {
		return "workspace", nil
	}
	if changeID == rows[0][1] {
		return "submitted", nil
	}
	if changeID == rows[0][0] {
		return "implemented", nil
	}
	return "", fmt.Errorf("change %s is not a current pointer", changeID)
}

type resolvedReviewFile struct {
	Review           reviewFile
	FileChangeID     string
	FileID           string
	ExistingReviewID string
}

func (s *doltStore) resolveReviewFile(state reviewState, f reviewFile) (resolvedReviewFile, error) {
	dir, name, err := resolveReviewPath(f.Path)
	if err != nil {
		return resolvedReviewFile{}, err
	}
	rows, err := s.queryCSV("SELECT fc.id,b.file_id FROM spec.filepaths p JOIN spec.dirs d ON d.id=p.dir_id JOIN spec.file_changes fc ON fc.filepath_id=p.id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(state.Workspace) + " AND w.parent_id=fc.change_id JOIN spec.bodies b ON b.file_id=fc.file_id AND b.state='workspace' WHERE d.path=" + sqlString(dir) + " AND p.name=" + sqlString(name) + " AND a.depth>0 AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(state.Workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) ORDER BY w.depth ASC LIMIT 1")
	if err != nil {
		return resolvedReviewFile{}, err
	}
	if len(rows) == 0 {
		return resolvedReviewFile{}, fmt.Errorf("file review path %q is not changed between submitted and workspace", f.Path)
	}
	if len(rows[0]) < 2 {
		return resolvedReviewFile{}, fmt.Errorf("could not resolve workspace body for %q", f.Path)
	}
	priorReviews, err := s.queryCSV("SELECT r.id FROM spec.file_change_reviews r JOIN spec.file_changes reviewed_fc ON reviewed_fc.id=r.file_change_id JOIN spec.changes_closure a ON a.child_id=reviewed_fc.change_id AND a.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(state.Workspace) + " AND w.parent_id=reviewed_fc.change_id WHERE reviewed_fc.file_id=" + rows[0][1] + " AND a.depth>0 ORDER BY w.depth ASC LIMIT 1")
	if err != nil {
		return resolvedReviewFile{}, err
	}
	var existingReviewID string
	if len(priorReviews) > 0 && len(priorReviews[0]) > 0 {
		existingReviewID = priorReviews[0][0]
	}
	return resolvedReviewFile{Review: f, FileChangeID: rows[0][0], FileID: rows[0][1], ExistingReviewID: existingReviewID}, nil
}

func resolveReviewPath(p string) (string, string, error) {
	p = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(p)), ".4sight/")
	if p == "." || strings.HasPrefix(p, "../") || filepath.IsAbs(p) {
		return "", "", fmt.Errorf("invalid review file path %q", p)
	}
	dir, name := filepath.ToSlash(filepath.Dir(p)), filepath.Base(p)
	if dir == "." {
		dir = ""
	}
	if !isCodeSpecName(name) && (dir != "schemas" || !strings.HasSuffix(name, ".md")) && !isTextAssetDir(dir) {
		return "", "", fmt.Errorf("unsupported review spec path %q", p)
	}
	return dir, name, nil
}

func (s *doltStore) nextReviewName() (string, error) {
	rows, err := s.queryCSV("SELECT name FROM spec.change_review")
	if err != nil {
		return "", err
	}
	used := map[string]bool{}
	for _, r := range rows {
		if len(r) > 0 {
			used[r[0]] = true
		}
	}
	for i := range 1000000 {
		name := fmt.Sprintf("r%011x", i)
		if len(name) > 12 {
			name = name[:12]
		}
		if !used[name] {
			return name, nil
		}
	}
	return "", fmt.Errorf("review name space exhausted")
}
