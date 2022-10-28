package net.tgburrin.pageview_svc;

public interface DatabaseRecordInterface<T> {
	public void validateRecord() throws InvalidDataException;
	public void mergeUpdate(T newRecord);
}
