using System.Threading;
using System.Threading.Tasks;

namespace Shop.Caching;

public interface IReadCache<T>
{
    ValueTask<T?> FindAsync(string key, CancellationToken ct = default);
    ValueTask SetAsync(string key, T value, CancellationToken ct = default);
}
