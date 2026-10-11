using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Shop.Data;

namespace Shop.Tests.Fakes;

public sealed class FakeRepository<T> : IRepository<T> where T : Entity
{
    private readonly ConcurrentDictionary<Guid, T> _items = new();

    public Task<T?> FindAsync(Guid id, CancellationToken ct = default)
        => Task.FromResult(_items.TryGetValue(id, out var item) ? item : null);

    public Task<IReadOnlyList<T>> ListAsync(CancellationToken ct = default)
        => Task.FromResult<IReadOnlyList<T>>(_items.Values.ToList());

    public Task AddAsync(T entity, CancellationToken ct = default)
    {
        _items[entity.Id] = entity;
        return Task.CompletedTask;
    }
}
