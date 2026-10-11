using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;

namespace Shop.Storage;

public sealed class BlobUploader
{
    private readonly HttpClient _client;
    private readonly Uri _baseUri;

    public BlobUploader(HttpClient client, Uri baseUri)
    {
        _client = client;
        _baseUri = baseUri;
    }

    public async Task UploadAsync(string blobName, byte[] content, CancellationToken cancellationToken = default)
    {
        var uri = new Uri(_baseUri, Uri.EscapeDataString(blobName));
        using var body = new ByteArrayContent(content);
        using var response = await _client.PutAsync(uri, body, cancellationToken).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();
    }
}
