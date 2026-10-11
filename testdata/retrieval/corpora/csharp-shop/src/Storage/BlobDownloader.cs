using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;

namespace Shop.Storage;

public sealed class BlobDownloader
{
    private readonly HttpClient _client;
    private readonly Uri _baseUri;

    public BlobDownloader(HttpClient client, Uri baseUri)
    {
        _client = client;
        _baseUri = baseUri;
    }

    public async Task<byte[]> DownloadAsync(string blobName, CancellationToken cancellationToken = default)
    {
        var uri = new Uri(_baseUri, Uri.EscapeDataString(blobName));
        using var response = await _client.GetAsync(uri, HttpCompletionOption.ResponseHeadersRead, cancellationToken)
            .ConfigureAwait(false);
        response.EnsureSuccessStatusCode();
        return await response.Content.ReadAsByteArrayAsync(cancellationToken).ConfigureAwait(false);
    }
}
