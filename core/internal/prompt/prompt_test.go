package prompt

import "testing"

func TestAdd_AutoVersion(t *testing.T) {
	r := NewRegistry()
	v1 := r.Add("greet", "Hello {{name}}", nil)
	v2 := r.Add("greet", "Hi {{name}}!", nil)
	if v1.Version != 1 || v2.Version != 2 {
		t.Errorf("versions: %d, %d", v1.Version, v2.Version)
	}
}

func TestLatest(t *testing.T) {
	r := NewRegistry()
	r.Add("greet", "v1", nil)
	r.Add("greet", "v2", nil)
	tmpl, err := r.Latest("greet")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Content != "v2" {
		t.Errorf("expected v2, got %s", tmpl.Content)
	}
}

func TestRender(t *testing.T) {
	result := Render("Hello {{name}}, you are {{role}}", map[string]string{
		"name": "Alice",
		"role": "admin",
	})
	if result != "Hello Alice, you are admin" {
		t.Errorf("got %q", result)
	}
}

func TestABTest(t *testing.T) {
	r := NewRegistry()
	a := r.Add("greet", "variant A", nil)
	b := r.Add("greet", "variant B", nil)
	r.AddABTest("test1", "greet", []string{a.ID, b.ID}, []int{100, 0})

	tmpl, err := r.Resolve("greet")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.ID != a.ID {
		t.Errorf("100%% weight on A, got %s", tmpl.ID)
	}
}
