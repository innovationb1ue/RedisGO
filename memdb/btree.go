package memdb

import (
	"fmt"
)

// Btree represents an AVL tree
type Btree[T Val] struct {
	root   *Node[T]
	values []T
	len    int
	dict   map[string]*Node[T]
}

// Val is the item in each Btree Node
type Val interface {
	Comp(val float64) int64
	GetNames() map[string]struct{}
	AddName(name string)
	DeleteName(name string)
	IsNameExist(name string) bool
}

// Node represents a node in the tree with a value, left and right children, and a height/balance of the node.
type Node[T Val] struct {
	Value       T
	left, right *Node[T]
	height      int64
}

// NewBtree returns a new btree
func NewBtree[T Val]() *Btree[T] { return new(Btree[T]).Init() }

// Init initializes all values/clears the tree and returns the tree pointer
func (t *Btree[T]) Init() *Btree[T] {
	t.root = nil
	t.values = nil
	t.len = 0
	t.dict = make(map[string]*Node[T])
	return t
}

// String returns a string representation of the tree values
func (t *Btree[T]) String() string {
	return fmt.Sprint(t.Values())
}

// Empty returns true if the tree is empty
func (t *Btree[T]) Empty() bool {
	return t.root == nil
}

// NotEmpty returns true if the tree is not empty
func (t *Btree[T]) NotEmpty() bool {
	return t.root != nil
}

func (t *Btree[T]) balance() int64 {
	if t.root != nil {
		return balance(t.root)
	}
	return 0
}

// Insert inserts a new value into the tree and returns the tree pointer
func (t *Btree[T]) Insert(r *record) bool {
	added := false
	t.root = insert[T](t.root, r, &added, t)
	if _, ok := t.dict[r.name]; !ok {
		added = true
	}
	if added {
		t.len++
	}
	// empty values
	t.values = nil
	return added
}

func insert[T Val](n *Node[T], r *record, added *bool, tree *Btree[T]) *Node[T] {
	// if node does not exist, create a new node
	if n == nil {
		*added = true
		n := (&Node[T]{Value: any(SortedSetNode{
			Names: map[string]struct{}{r.name: {}},
			Score: r.score,
		}).(T)}).Init()
		return n
	}
	// compare target with current node by score
	c := n.Value.Comp(r.score)
	if c < 0 {
		n.right = insert(n.right, r, added, tree)
	} else if c > 0 {
		n.left = insert(n.left, r, added, tree)
	} else {
		// if node exist, we add names to this node
		for name, _ := range n.Value.GetNames() {
			n.Value.AddName(name)
		}
		*added = false
		return n
	}

	n.height = n.maxHeight() + 1
	c = balance(n)

	if c > 1 {
		c = n.Value.Comp(r.score)
		if c > 0 {
			return n.rotateRight()
		} else if c < 0 {
			n.left = n.left.rotateLeft()
			return n.rotateRight()
		}
	} else if c < -1 {
		c = n.Value.Comp(r.score)
		if c < 0 {
			return n.rotateLeft()
		} else if c > 0 {
			n.right = n.right.rotateRight()
			return n.rotateLeft()
		}
	}
	return n
}

// InsertAll inserts all the values into the tree and returns the tree pointer
func (t *Btree[T]) InsertAll(values []*record) *Btree[T] {
	for _, v := range values {
		t.Insert(v)
	}
	return t
}

// Contains returns true if the tree contains the specified value
func (t *Btree[T]) Contains(value Val) bool {
	return any(t.Get(value)) != nil
}

// ContainsAny returns true if the tree contains any of the values
func (t *Btree[T]) ContainsAny(values []Val) bool {
	for _, v := range values {
		if t.Contains(v) {
			return true
		}
	}
	return false
}

// ContainsAll returns true if the tree contains all of the values
func (t *Btree[T]) ContainsAll(values []Val) bool {
	for _, v := range values {
		if !t.Contains(v) {
			return false
		}
	}
	return true
}

// Get returns the node value associated with the search value
func (t *Btree[T]) Get(value Val) T {
	var node *Node[T]
	if t.root != nil {
		node = t.root.get(value)
	}
	if node != nil {
		return node.Value
	}
	return nil
}

func (t *Btree[T]) GetByName(name string) *Node[T] {
	var node *Node[T]
	node, ok := t.dict[name]
	if !ok {
		return nil
	} else {
		return node
	}
}

// Len return the number of nodes in the tree
func (t *Btree[T]) Len() int {
	return t.len
}

// Head returns the first value in the tree
func (t *Btree[T]) Head() Val {
	if t.root == nil {
		return nil
	}
	var beginning = t.root
	for beginning.left != nil {
		beginning = beginning.left
	}
	if beginning == nil {
		for beginning.right != nil {
			beginning = beginning.right
		}
	}
	if beginning != nil {
		return beginning.Value
	}
	return nil
}

// Tail returns the last value in the tree
func (t *Btree[T]) Tail() Val {
	if t.root == nil {
		return nil
	}
	var beginning = t.root
	for beginning.right != nil {
		beginning = beginning.right
	}
	if beginning == nil {
		for beginning.left != nil {
			beginning = beginning.left
		}
	}
	if beginning != nil {
		return beginning.Value
	}
	return nil
}

// Values returns a slice of all the values in tree in order
func (t *Btree[T]) Values() []T {
	if t.values == nil {
		t.values = make([]T, t.len)
		t.Ascend(func(n *Node[T], i int) bool {
			t.values[i] = n.Value
			return true
		})
	}
	return t.values
}

// Delete deletes the node from the tree associated with the search value
func (t *Btree[T]) Delete(value string) *Btree[T] {
	deleted := false
	t.root = deleteNode(t.root, value, &deleted)
	if deleted {
		t.len--
	}
	t.values = nil
	return t
}

// DeleteAll deletes the nodes from the tree associated with the search values
func (t *Btree[T]) DeleteAll(values []Val) *Btree[T] {
	for _, v := range values {
		t.Delete(v)
	}
	return t
}

func rebalance[T Val](n *Node[T]) *Node[T] {
	//re-balance
	if n == nil {
		return n
	}
	n.height = n.maxHeight() + 1
	bal := balance(n)
	if bal > 1 {
		if balance(n.left) >= 0 {
			return n.rotateRight()
		}
		n.left = n.left.rotateLeft()
		return n.rotateRight()
	} else if bal < -1 {
		if balance(n.right) <= 0 {
			return n.rotateLeft()
		}
		n.right = n.right.rotateRight()
		return n.rotateLeft()
	}
	return n
}

func deleteNode[T Val](n *Node[T], target Val, deleted *bool) *Node[T] {
	if n == nil {
		return n
	}
	// compare the node target.
	c := target.Comp()

	// target value smaller than current node value
	if c < 0 {
		n.left = deleteNode(n.left, target, deleted)
	} else if c > 0 {
		n.right = deleteNode(n.right, target, deleted)
	} else {
		// goes into this condition means we find the node we want to delete.
		*deleted = true
		// nothing smaller than this node
		if n.left == nil {
			t := n.right
			n.Init() // remove pointers in this node
			return t
		}
		// nothing larger than this node
		if n.right == nil {
			t := n.left
			n.Init() // remove pointers in this node
			return t
		}
		// intermediate node. we set the minimum child of right child to this node
		t := n.right.min()
		n.Value = t.Value
		n.right = deleteNode(n.right, t.Value, deleted)
	}

	//re-balance
	n = rebalance(n)
	return n
}

// Pop deletes the last node from the tree and returns its value
func (t *Btree[T]) Pop() Val {
	value := t.Tail()
	if value != nil {
		t.Delete(value)
	}
	return value
}

// Pull deletes the first node from the tree and returns its value
func (t *Btree[T]) Pull() Val {
	value := t.Head()
	if value != nil {
		t.Delete(value)
	}
	return value
}

// NodeIterator expresses the iterator function used for traversals
type NodeIterator[T Val] func(n *Node[T], i int) bool

// Ascend performs an ascending order traversal of the tree calling the iterator function on each node
// the iterator will continue as long as the NodeIterator returns true
func (t *Btree[T]) Ascend(iterator NodeIterator[T]) {
	var i int
	if t.root != nil {
		t.root.iterate(iterator, &i, true)
	}
}

// Descend performs a descending order traversal of the tree using the iterator
// the iterator will continue as long as the NodeIterator returns true
func (t *Btree[T]) Descend(iterator NodeIterator[T]) {
	var i int
	if t.root != nil {
		t.root.rIterate(iterator, &i, true)
	}
}

// Debug prints out useful debug information about the tree for debugging purposes
func (t *Btree[T]) Debug() {
	fmt.Println("----------------------------------------------------------------------------------------------")
	if t.Empty() {
		fmt.Println("tree is empty")
	} else {
		fmt.Println(t.Len(), "elements")
	}

	t.Ascend(func(n *Node[T], i int) bool {
		if t.root.Value.Comp(n.Value) == 0 {
			fmt.Print("ROOT ** ")
		}
		n.Debug()
		return true
	})
	fmt.Println("----------------------------------------------------------------------------------------------")
}

// Init initializes the values of the node or clears the node and returns the node pointer
func (n *Node[T]) Init() *Node[T] {
	n.height = 1
	n.left = nil
	n.right = nil
	return n
}

// String returns a string representing the node
func (n *Node[T]) String() string {
	return fmt.Sprint(n.Value)
}

// Debug prints out useful debug information about the tree node for debugging purposes
func (n *Node[T]) Debug() {
	var children string
	if n.left == nil && n.right == nil {
		children = "no children |"
	} else if n.left != nil && n.right != nil {
		children = fmt.Sprint("left child:", n.left.String(), " right child:", n.right.String())
	} else if n.right != nil {
		children = fmt.Sprint("right child:", n.right.String())
	} else {
		children = fmt.Sprint("left child:", n.left.String())
	}

	fmt.Println(n.String(), "|", "height", n.height, "|", "balance", balance(n), "|", children)
}

func height[T Val](n *Node[T]) int64 {
	if n != nil {
		return n.height
	}
	return 0
}

func balance[T Val](n *Node[T]) int64 {
	if n == nil {
		return 0
	}
	return height(n.left) - height(n.right)
}

// get return the node based on the value
func (n *Node[T]) get(target Val) *Node[T] {
	var node *Node[T]
	c := target.Comp(n.Value)
	// target value < current value
	if c < 0 {
		if n.left != nil {
			node = n.left.get(target)
		}
	} else if c > 0 {
		if n.right != nil {
			node = n.right.get(target)
		}
	} else {
		node = n
	}
	return node
}

func (n *Node[T]) rotateRight() *Node[T] {
	l := n.left
	// Rotation
	l.right, n.left = n, l.right

	// update heights
	n.height = n.maxHeight() + 1
	l.height = l.maxHeight() + 1

	return l
}

func (n *Node[T]) rotateLeft() *Node[T] {
	r := n.right
	// Rotation
	r.left, n.right = n, r.left

	// update heights
	n.height = n.maxHeight() + 1
	r.height = r.maxHeight() + 1

	return r
}

func (n *Node[T]) iterate(iterator NodeIterator[T], i *int, cont bool) {
	if n != nil && cont {
		n.left.iterate(iterator, i, cont)
		cont = iterator(n, *i)
		*i++
		n.right.iterate(iterator, i, cont)
	}
}

func (n *Node[T]) rIterate(iterator NodeIterator[T], i *int, cont bool) {
	if n != nil && cont {
		n.right.iterate(iterator, i, cont)
		cont = iterator(n, *i)
		*i++
		n.left.iterate(iterator, i, cont)
	}
}

func (n *Node[T]) min() *Node[T] {
	current := n
	for current.left != nil {
		current = current.left
	}
	return current
}

func (n *Node[T]) maxHeight() int64 {
	rh := height(n.right)
	lh := height(n.left)
	if rh > lh {
		return rh
	}
	return lh
}
