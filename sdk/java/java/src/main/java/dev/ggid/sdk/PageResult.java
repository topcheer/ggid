package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

/**
 * Paginated result wrapper.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PageResult<T> {
    public List<T> items;

    @JsonProperty("total_count")
    public int totalCount;

    public int page;

    @JsonProperty("page_size")
    public int pageSize;

    public List<T> getItems() { return items; }
    public int getTotalCount() { return totalCount; }
    public int getPage() { return page; }
    public int getPageSize() { return pageSize; }
}
