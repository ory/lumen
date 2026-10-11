using System;
using System.Net;
using System.Threading.Tasks;

namespace Shop.Storage;

[Obsolete("Use BlobDownloader")]
public class LegacyBlobClient
{
    private readonly string _baseUrl;

    public LegacyBlobClient(string baseUrl) => _baseUrl = baseUrl;

    public Task<byte[]> DownloadAsync(string blobName)
    {
        using var web = new WebClient();
        return web.DownloadDataTaskAsync(_baseUrl + "/" + blobName);
    }
}
