package memdb

import (
	"errors"
	"fmt"
)

// Btree represents an AVL tree
type Btree[T Val] struct {
	root   *Node[T]
	values []map[string]float64
	len    int
	dict   map[string]*Node[T]
}

// Val is the item in each Btree Node
type Val interface {
	Comp(val Val) int64 // compare value with another Val
	SetScore(score float64)
	GetScore() float64
	GetNames() map[string]struct{} // allowed multiple names in a single node
	AddName(name string)
	DeleteName(name string)
	Empty()
	IsNameExist(name string) bool
}

// Node represents a node in the tree with a value, left and right children, and a height/balance of the node.
type Node[T Val] struct {
	Value       T
	left, right *Node[T]
	height      int64
}

// NewBtree returns a new btree
func NewBtree[T Val]() *Btree[T] {
	return new(Btree[T]).Init()
}

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
func (t *Btree[T]) Insert(r T) bool {
	nodeAdded := false
	// do insertion, might change the structure of tree
	t.root = insert[T](t.root, r, &nodeAdded, t)
	if nodeAdded {
		t.len++
	}
	// empty values
	t.values = nil
	return nodeAdded
}

func insert[T Val](n *Node[T], target T, added *bool, tree *Btree[T]) *Node[T] {
	// if this node does not exist, create a new node
	if n == nil {
		*added = true
		node := (&Node[T]{Value: target}).Init()
		for name := range target.GetNames() {
			tree.dict[name] = node
		}
		return node
	}
	// compare current with target node by score
	c := n.Value.Comp(target)
	if c < 0 {
		n.right = insert(n.right, target, added, tree)
	} else if c > 0 {
		n.left = insert(n.left, target, added, tree)
	} else {
		// if node exist, we add names to this node
		for name := range target.GetNames() {
			n.Value.AddName(name)
			tree.dict[name] = n
		}
		// didn't create new node
		*added = false
		return n
	}

	n.height = n.maxHeight() + 1
	c = balance(n)

	if c > 1 {
		c = n.Value.Comp(target)
		if c > 0 {
			return n.rotateRight()
		} else if c < 0 {
			n.left = n.left.rotateLeft()
			return n.rotateRight()
		}
	} else if c < -1 {
		c = n.Value.Comp(target)
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
func (t *Btree[T]) InsertAll(values []T) *Btree[T] {
	for _, v := range values {
		t.Insert(v)
	}
	return t
}

// Contains returns true if the tree contains the specified value
func (t *Btree[T]) Contains(value T) bool {
	_, ok := t.Get(value)
	return ok
}

// ContainsAny returns true if the tree contains any of the values
func (t *Btree[T]) ContainsAny(values []T) bool {
	for _, v := range values {
		if t.Contains(v) {
			return true
		}
	}
	return false
}

// ContainsAll returns true if the tree contains all the values
func (t *Btree[T]) ContainsAll(values []T) bool {
	for _, v := range values {
		if !t.Contains(v) {
			return false
		}
	}
	return true
}

// Get returns the node value associated with the search value
func (t *Btree[T]) Get(value T) (T, bool) {
	var node *Node[T]
	if t.root != nil {
		node = t.root.get(value)
	}
	if node != nil {
		return node.Value, true
	}
	return *new(T), false
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
func (t *Btree[T]) Tail() (T, error) {
	if t.root == nil {
		return *new(T), errors.New("")
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
		return beginning.Value, nil
	}
	return *new(T), errors.New("")
}

// Values returns a slice of all the values in tree in order
func (t *Btree[T]) Values() []map[string]float64 {
	if t.values == nil {
		t.values = make([]map[string]float64, 0)
		t.Ascend(func(n *Node[T], i int) bool {
			for name := range n.Value.GetNames() {
				t.values = append(t.values, map[string]float64{name: n.Value.GetScore()})
			}
			return true
		})
	}
	return t.values
}

// Delete deletes the node from the tree associated with the search value
func (t *Btree[T]) Delete(name string) *Btree[T] {
	deleted := false
	// get node by name
	node, ok := t.dict[name]
	// deleting a non-exist member
	if !ok {
		return nil
	}
	// delete it
	t.root = deleteNode(t.root, node.Value, &deleted)
	delete(t.dict, name)
	if deleted {
		t.len--
	}
	t.values = nil
	return t
}

// DeleteAll deletes the nodes from the tree associated with the search values
func (t *Btree[T]) DeleteAll(values []string) *Btree[T] {
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

func deleteNode[T Val](n *Node[T], target T, deleted *bool) *Node[T] {
	if n == nil {
		return n
	}
	// compare the node target by score
	c := target.Comp(n.Value)

	// target value smaller than current node value
	if c < 0 {
		n.left = deleteNode(n.left, target, deleted)
	} else if c > 0 {
		n.right = deleteNode(n.right, target, deleted)
	} else {
		// goes into this condition means we find the node we want to delete.
		// if no member in this node anymore or we want to remove all members.
		if len(n.Value.GetNames()) == 0 || len(target.GetNames()) == len(n.Value.GetNames()) {
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
			// till here we have an intermediate node. we set the minimum child of right child to this node
			t := n.right.min()
			// note here. Value is a pointer. We point this node Value to the child Value
			// but the pointers point to that child will be deleted so that child become unreachable and can be freed
			n.Value = t.Value
			// delete that child
			n.right = deleteNode(n.right, t.Value, deleted)
			return n
		} else {
			// remove a subset of names
			for name := range target.GetNames() {
				n.Value.DeleteName(name)
			}
			// still has members in this node, we just return it. no tree rotation needed since no change of structure
			return n
		}
	}
	//re-balance
	n = rebalance(n)
	return n
}

// Pop deletes the last node from the tree and returns its value
func (t *Btree[T]) Pop() (T, error) {
	value, err := t.Tail()
	if err != nil {
		return value, err
	}
	for name := range value.GetNames() {
		t.Delete(name)
		break
	}
	return value, nil
}

// Pull deletes the first node from the tree and returns its value
func (t *Btree[T]) Pull() Val {
	value := t.Head()
	if value != nil {
		for name := range value.GetNames() {
			t.Delete(name)
			break
		}
	}
	return value
}

// NodeIterator expresses the iterator function used for traversals
type NodeIterator[T Val] func(node *Node[T], i int) bool

// Ascend performs an ascending order traversal of the tree calling the iterator function on each node
// the iterator will continue as long as the NodeIterator returns true
func (t *Btree[T]) Ascend(iterator NodeIterator[T]) {
	var i int
	if t.root != nil {
		t.root.iterate(iterator, &i, true)
	}
}

// Descend performs a descending order traversal of the tree using the iterator
//
//	will continue as long as the NodeIterator returns true
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
func (n *Node[T]) get(target T) *Node[T] {
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
