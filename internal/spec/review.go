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
type reviewFile struct{ Path, Why, Summary, Description, Details, BodySummary string }
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
	paths, err := st.unreviewedPaths(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	if state.Prompt == "" && len(paths) == 0 {
		_, err = fmt.Fprintln(out, meta+"There is nothing to review.")
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
	specs := make([]string, 0, len(bodies))
	for _, d := range bodies {
		specs = append(specs, specRelPath(d.Dir, d.Name))
	}
	slices.Sort(specs)
	removed, err := st.removedSpecPaths()
	if err != nil {
		return err
	}
	schemas, err := listSchemaSpecs(filepath.Join(root, "schemas"))
	if err != nil {
		return err
	}
	projectRoot := filepath.Dir(root)
	var b strings.Builder
	b.WriteString("Review the current unsubmitted specification changes for quality and consistency.\n\n")
	b.WriteString(meta)
	b.WriteString("\nWorkspace Specs\n")
	for _, p := range specs {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	b.WriteString("\nRemoved Specs\n")
	if len(removed) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, p := range removed {
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
	return runChangeReviewResponse(root, newDoltStore(filepath.Join(root, "db")), response)
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
	expected, err := st.unreviewedPaths(state.Submitted, state.Workspace)
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
			return fmt.Errorf("file review path %q is not an unreviewed change between submitted and workspace", f.Path)
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
	for _, f := range r.Files {
		if err := st.persistFileReview(state, f); err != nil {
			return err
		}
	}
	score := 0
	if r.Ready {
		score = 100
	}
	changeName, err := st.nextReviewName()
	if err != nil {
		return err
	}
	if err := st.run("sql", "-q", fmt.Sprintf("INSERT INTO spec.change_review (name,change_id,why,summary,description,details,score,review) VALUES (%s,%s,%s,%s,%s,%s,%d,%s)", sqlString(changeName), sqlString(state.Workspace), sqlBodyLiteral(r.Why), sqlBodyLiteral(r.Summary), sqlBodyLiteral(r.Description), sqlBodyLiteral(r.Details), score, sqlBodyLiteral(r.Review))); err != nil {
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

func (s *doltStore) removedSpecPaths() ([]string, error) {
	rows, err := s.queryCSV("SELECT d.path,f.name FROM spec.files f JOIN spec.dirs d ON d.id=f.dir_id LEFT JOIN spec.bodies bs ON bs.file_id=f.id AND bs.state='submitted' LEFT JOIN spec.bodies bw ON bw.file_id=f.id AND bw.state='workspace' WHERE COALESCE(bs.body,'')<>'' AND COALESCE(bw.body,'')='' ORDER BY d.path,f.name")
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		paths = append(paths, specRelPath(row[0], row[1]))
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

func (s *doltStore) persistFileReview(state reviewState, f reviewFile) error {
	dir, name, err := resolveReviewPath(f.Path)
	if err != nil {
		return err
	}
	rows, err := s.queryCSV("SELECT fc.id,b.body FROM spec.dirs d JOIN spec.files x ON x.dir_id=d.id JOIN spec.file_changes fc ON fc.file_id=x.id JOIN spec.changes_closure a ON a.child_id=fc.change_id AND a.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure w ON w.child_id=" + sqlString(state.Workspace) + " AND w.parent_id=fc.change_id JOIN spec.bodies b ON b.file_id=x.id AND b.state='workspace' WHERE d.path=" + sqlString(dir) + " AND x.name=" + sqlString(name) + " AND a.depth>0 AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure na ON na.child_id=newer.change_id AND na.parent_id=" + sqlString(state.Submitted) + " JOIN spec.changes_closure nw ON nw.child_id=" + sqlString(state.Workspace) + " AND nw.parent_id=newer.change_id WHERE newer.file_id=fc.file_id AND na.depth>0 AND nw.depth<w.depth) ORDER BY w.depth ASC LIMIT 1")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("file review path %q is not changed between submitted and workspace", f.Path)
	}
	if len(rows[0]) < 2 {
		return fmt.Errorf("could not resolve workspace body for %q", f.Path)
	}
	stmt := fmt.Sprintf("INSERT INTO spec.file_change_reviews(file_change_id,why,summary,description,details) VALUES(%s,%s,%s,%s,%s)", rows[0][0], sqlBodyLiteral(f.Why), sqlBodyLiteral(f.Summary), sqlBodyLiteral(f.Description), sqlBodyLiteral(f.Details))
	if err := s.run("sql", "-q", stmt); err != nil {
		return fmt.Errorf("save file review for %s: %w", f.Path, err)
	}
	if err := s.run("sql", "-q", fmt.Sprintf("UPDATE spec.bodies b JOIN spec.files x ON x.id=b.file_id JOIN spec.dirs d ON d.id=x.dir_id SET b.summary=%s WHERE b.state='workspace' AND d.path=%s AND x.name=%s", sqlBodyLiteral(f.BodySummary), sqlString(dir), sqlString(name))); err != nil {
		return fmt.Errorf("save body summary for %s: %w", f.Path, err)
	}
	return nil
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
