<!-- Source: https://docs.djangoproject.com/en/5.1/ref/models/querysets/ -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Django's ORM uses lazy QuerySets that don't execute database queries until
evaluated. QuerySets are created via a Manager (the default interface from
Model to the database). Filtering is done through methods like `filter()` and
`exclude()`, which return new QuerySets (chaining via internal cloning). The
`Query` class (in `django-sql-query.py`) compiles the filter tree into SQL.
Complex lookups use `Q` objects (from `django-q.py`) which can be combined
with `&` (AND) and `|` (OR) operators. The Manager class provides the initial
QuerySet via `get_queryset()`.

## Key Types in Fixtures

**django-query.py — QuerySet and iterables:**
- `QuerySet` — main lazy query class, extends `AltersData`
- `BaseIterable` — base class for iteration strategies
- `ModelIterable` — iterates as model instances (default)
- `RawModelIterable` — iterates raw query results as model instances
- `ValuesIterable` — iterates as dictionaries
- `ValuesListIterable` — iterates as tuples
- `NamedValuesListIterable` — iterates as named tuples
- `FlatValuesListIterable` — iterates as flat values
- `EmptyQuerySet` — singleton empty queryset (via `InstanceCheckMeta`)
- `InstanceCheckMeta` — metaclass for EmptyQuerySet isinstance checks

**django-sql-query.py — SQL compilation:**
- `Query` — represents SQL query, extends `BaseExpression`
- `RawQuery` — raw SQL query wrapper
- `JoinPromoter` — manages JOIN type promotion

**django-manager.py — Manager layer:**
- `BaseManager` — base manager providing QuerySet creation
- `Manager` — default manager, extends `BaseManager.from_queryset(QuerySet)`
- `ManagerDescriptor` — descriptor for model.objects access
- `EmptyManager` — manager that always returns empty QuerySet

**django-q.py — Complex lookups:**
- `Q` — query filter node, extends `tree.Node`, supports `&` and `|` operators
- `DeferredAttribute` — descriptor for deferred model field loading
- `RegisterLookupMixin` — mixin for registering custom lookups
- `FilteredRelation` — represents filtered JOINs

## Required Facts

1. `QuerySet` does NOT execute queries until evaluated — evaluation is triggered by iteration (`__iter__`), `len()`, `list()`, `bool()`, slicing with step, or `repr()`.
2. `QuerySet.__iter__()` calls `self._fetch_all()` which populates `self._result_cache` — subsequent iterations use the cache.
3. `QuerySet.filter()` and `QuerySet.exclude()` return new QuerySet instances via `self._filter_or_exclude()` — they do NOT modify the original.
4. `QuerySet` chaining works through `self._clone()` which creates a copy of the QuerySet with the same underlying `Query`.
5. `QuerySet.all()` returns a copy of the current QuerySet (via `_clone()`).
6. `QuerySet` has both public filtering methods (`filter`, `exclude`, `annotate`, `order_by`, `values`, `values_list`, `distinct`, `select_related`, `prefetch_related`) and internal methods (`_filter_or_exclude`, `complex_filter`).
7. `Query` class (django-sql-query.py) extends `BaseExpression` and represents the SQL query — it tracks WHERE clauses, JOINs, aggregates, and ordering.
8. `Manager` extends `BaseManager.from_queryset(QuerySet)` — this dynamically creates a class that proxies QuerySet methods through the manager.
9. `BaseManager.get_queryset()` returns a new `QuerySet(self.model, using=self._db, hints=self._hints)`.
10. `Q` extends `tree.Node` and implements `__or__`, `__and__`, and `__invert__` for combining filters with `|`, `&`, and `~` operators.
11. `Q` has a `connector` attribute (default `AND`) and `negated` attribute for NOT queries.
12. `QuerySet` stores its SQL representation in `self.query` which is a `Query` instance.
13. `QuerySet.complex_filter()` handles both `Q` objects and plain keyword lookups.
14. `ModelIterable` is the default `_iterable_class` on QuerySet — it creates model instances from database rows.
15. `EmptyQuerySet` uses `InstanceCheckMeta` metaclass to support `isinstance()` checks without actual instantiation.
16. `BaseManager` uses `_get_queryset_methods()` to dynamically copy QuerySet methods onto the manager class.
17. `QuerySet` supports slicing via `__getitem__` which modifies the `Query` with LIMIT/OFFSET.

## Hallucination Traps

- There is NO `Prefetch` class in the fixtures (it exists in Django but not in this fixture set).
- There is NO `F` expression class in the fixtures.
- There is NO `Subquery` class in the fixtures.
- There is NO `RawQuerySet` class in the fixtures — there is `RawQuery` (in django-sql-query.py) and `RawModelIterable` (in django-query.py), but not `RawQuerySet`.
- `QuerySet` does NOT extend `list` — it extends `AltersData`.
- `Query` does NOT extend `QuerySet` — it extends `BaseExpression`. They are separate classes.
- `Manager` does NOT define `filter()` or `exclude()` directly — these are dynamically proxied from QuerySet via `from_queryset()`.
- There is NO `Lookup` class definition in the fixtures — only `RegisterLookupMixin`.
- `Q` extends `tree.Node`, NOT `Expression` or `BaseExpression`.
