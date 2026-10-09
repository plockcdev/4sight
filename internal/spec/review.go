package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

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

type changeReviewFileReport struct {
	Path, Why, Summary, Description, Details, BodySummary string
}

func runChangeReviewShow(st *doltStore, changeID string, out io.Writer) error {
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
	reviews, err := st.queryCSV("SELECT why,summary,description,details,review,ready FROM spec.change_review WHERE change_id=" + sqlString(changeID))
	if err != nil {
		return fmt.Errorf("look up review for change %s: %w", changeID, err)
	}
	if len(reviews) == 0 {
		return fmt.Errorf("no review is available for change %s", changeID)
	}
	if len(reviews[0]) < 6 {
		return fmt.Errorf("incomplete review data for change %s", changeID)
	}
	rows, err := st.queryCSV("SELECT d.path,f.name,r.why,r.summary,r.description,r.details,r.body_summary FROM spec.file_change_reviews r JOIN spec.change_review cr ON cr.id=r.change_review_id JOIN spec.file_changes fc ON fc.id=r.file_change_id JOIN spec.files f ON f.id=fc.file_id JOIN spec.dirs d ON d.id=f.dir_id WHERE cr.change_id=" + sqlString(changeID))
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
	if _, err := fmt.Fprintf(out, "Change ID: %s\n\nOverall review:\n", changeID); err != nil {
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
	meta := fmt.Sprintf("Workspace change: %s\nSubmitted change: %s\nChange prompt: %s\n", state.Workspace, state.Submitted, nonempty(state.Prompt, "(empty)"))
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
	unreviewedPaths, err := st.unreviewedPaths(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	if state.Prompt == "" && len(unreviewedPaths) == 0 {
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
		if normalizeSpecBody(d.Body) == "" {
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
	b.WriteString(meta)
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
	fmt.Fprintf(&b, "\nComposite diff: %s\n\nRead the composite diff to review all changes since the last submitted specification. Read the full workspace spec for every file referenced in the diff, read all database schema specs listed above, and read workspace specs related to changed files or the change prompt, including parent command design docs and coding standards as applicable.\n", diffProjectPath)
	responseSchema := filepath.ToSlash(filepath.Join(".4sight", "subcommands", "spec", "subcommands", "change", "subcommands", "review", "subcommands", "agent", "responseSchema.json"))
	fmt.Fprintf(&b, "\nRead the response schema at %s and follow its constraints.\n", responseSchema)
	b.WriteString("\nChanged spec files to review:\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	b.WriteString("\nSubmit a qualitative review of the current workspace change compared with the last submitted specification by sending one JSON object to `4sight spec change review agent response` on stdin. Include an overall change review and one file_reviews entry for every file updated since submitted. Each file entry needs path, why, summary (max 65 characters), description, details, and body_summary (max 65 characters). Include overall why, summary (max 65 characters), description, details, review, and ready boolean. JSON shape:\n")
	b.WriteString("```json\n{\"why\":\"...\",\"summary\":\"...\",\"description\":\"...\",\"details\":\"...\",\"review\":\"...\",\"ready\":true,\"file_reviews\":[{\"path\":\".4sight/subcommands/example/cmd.md\",\"why\":\"...\",\"summary\":\"...\",\"description\":\"...\",\"details\":\"...\",\"body_summary\":\"...\"}]}\n```\n")
	if state.Prompt != "" {
		b.WriteString("\n---\nUser request prompt (make any resulting changes to specs on disk, creating command directories or spec files as needed):\n")
		b.WriteString(state.Prompt)
		b.WriteString("\n\nIf you changed any specs because of this request, run `4sight spec change import` to record them, then rerun `4sight spec change review agent prompt` and follow the refreshed instructions before reviewing. If no spec changed, tell the user no review is needed for the prompt because it caused no spec changes, then stop.\n")
	}
	_, err = io.WriteString(out, b.String())
	return err
}

func nonempty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func ChangeReviewAgentResponse(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
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
	st := newDoltStore(filepath.Join(root, "db"))
	state, err := st.reviewState()
	if err != nil {
		return err
	}
	if err := runChangeReviewResponse(root, st, response); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "Saved review for change %s.\n", state.Workspace)
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

func runChangeReviewResponse(root string, st *doltStore, r reviewResponse) error {
	if err := validateReview(r); err != nil {
		return err
	}
	state, err := st.reviewState()
	if err != nil {
		return err
	}
	exists, err := st.queryCSV("SELECT id FROM spec.change_review WHERE change_id = " + sqlString(state.Workspace))
	if err != nil {
		return err
	}
	if len(exists) > 0 {
		return fmt.Errorf("workspace change %s already has a review; submit a new request prompt or import a spec change to create a new workspace change", state.Workspace)
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
	fmt.Fprintf(&tx, "INSERT INTO spec.change_review (name,change_id,why,summary,description,details,ready,review) VALUES (%s,%s,%s,%s,%s,%s,%d,%s);\n", sqlString(changeName), sqlString(state.Workspace), sqlBodyLiteral(r.Why), sqlBodyLiteral(r.Summary), sqlBodyLiteral(r.Description), sqlBodyLiteral(r.Details), ready, sqlBodyLiteral(r.Review))
	for _, file := range resolved {
		if file.ExistingReviewID != "" {
			fmt.Fprintf(&tx, "UPDATE spec.file_change_reviews SET file_change_id=%s,change_review_id=(SELECT id FROM spec.change_review WHERE change_id=%s),why=%s,summary=%s,body_summary=%s,description=%s,details=%s WHERE id=%s;\n", file.FileChangeID, sqlString(state.Workspace), sqlBodyLiteral(file.Review.Why), sqlBodyLiteral(file.Review.Summary), sqlBodyLiteral(file.Review.BodySummary), sqlBodyLiteral(file.Review.Description), sqlBodyLiteral(file.Review.Details), file.ExistingReviewID)
		} else {
			fmt.Fprintf(&tx, "INSERT INTO spec.file_change_reviews (file_change_id,change_review_id,why,summary,body_summary,description,details) VALUES (%s,(SELECT id FROM spec.change_review WHERE change_id=%s),%s,%s,%s,%s,%s);\n", file.FileChangeID, sqlString(state.Workspace), sqlBodyLiteral(file.Review.Why), sqlBodyLiteral(file.Review.Summary), sqlBodyLiteral(file.Review.BodySummary), sqlBodyLiteral(file.Review.Description), sqlBodyLiteral(file.Review.Details))
		}
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
		db[specKey{d.Dir, d.Name}] = normalizeSpecBody(d.Body)
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
			diskText = normalizeSpecBody(body)
		}
		dbText := db[k]
		if dbText == diskText {
			continue
		}
		plus, minus := lineChangeCounts(dbText, diskText)
		out = append(out, fmt.Sprintf("%s is out of sync (+%d/-%d on disk vs workspace body)", specRelPath(k.dir, k.name), plus, minus))
	}
	slices.Sort(out)
	return out, nil
}

func (s *doltStore) unreviewedPaths(submitted, workspace string) ([]string, error) {
	rows, err := s.queryCSV("SELECT d.path,f.name FROM spec.file_changes fc JOIN spec.files f ON f.id=fc.file_id JOIN spec.dirs d ON d.id=f.dir_id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(workspace) + " AND w.parent_id=fc.change_id LEFT JOIN spec.file_change_reviews r ON r.file_change_id=fc.id WHERE a.depth>0 AND r.id IS NULL AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) GROUP BY d.path,f.name ORDER BY d.path,f.name")
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
	rows, err := s.queryCSV("SELECT d.path,f.name FROM spec.file_changes fc JOIN spec.files f ON f.id=fc.file_id JOIN spec.dirs d ON d.id=f.dir_id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(workspace) + " AND w.parent_id=fc.change_id WHERE a.depth>0 AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) GROUP BY d.path,f.name ORDER BY d.path,f.name")
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

func (s *doltStore) removedSpecPaths() ([]specKey, error) {
	rows, err := s.queryCSV("SELECT d.path,f.name,COALESCE(bs.body,''),COALESCE(bw.body,'') FROM spec.files f JOIN spec.dirs d ON d.id=f.dir_id LEFT JOIN spec.bodies bs ON bs.file_id=f.id AND bs.state='submitted' LEFT JOIN spec.bodies bw ON bw.file_id=f.id AND bw.state='workspace' ORDER BY d.path,f.name")
	if err != nil {
		return nil, err
	}
	paths := make([]specKey, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 || normalizeSpecBody(row[2]) == "" || normalizeSpecBody(row[3]) != "" {
			continue
		}
		paths = append(paths, specKey{dir: row[0], name: row[1]})
	}
	return paths, nil
}

func (s *doltStore) compositeDiff(submitted, workspace string) (string, error) {
	rows, err := s.queryCSV("SELECT d.path,f.name,COALESCE(MAX(CASE WHEN b.state='submitted' THEN b.body ELSE NULL END),''),COALESCE(MAX(CASE WHEN b.state='workspace' THEN b.body ELSE NULL END),'') FROM spec.files f JOIN spec.dirs d ON d.id=f.dir_id JOIN spec.bodies b ON b.file_id=f.id AND b.state IN ('submitted','workspace') GROUP BY d.path,f.name ORDER BY d.path,f.name")
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, r := range rows {
		if len(r) < 4 {
			continue
		}
		old, new := r[2], r[3]
		if old == new {
			continue
		}
		p := ".4sight/" + specRelPath(r[0], r[1])
		diff := unifiedDiffPaths(p, p, old, new)
		if diff != "" {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(diff)
		}
	}
	return out.String(), nil
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
	rows, err := s.queryCSV("SELECT fc.id,b.file_id FROM spec.dirs d JOIN spec.files x ON x.dir_id=d.id JOIN spec.file_changes fc ON fc.file_id=x.id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(state.Workspace) + " AND w.parent_id=fc.change_id JOIN spec.bodies b ON b.file_id=x.id AND b.state='workspace' WHERE d.path=" + sqlString(dir) + " AND x.name=" + sqlString(name) + " AND a.depth>0 AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(state.Workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) ORDER BY w.depth ASC LIMIT 1")
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
	if !isCodeSpecName(name) && (dir != "schemas" || !strings.HasSuffix(name, ".md")) {
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
