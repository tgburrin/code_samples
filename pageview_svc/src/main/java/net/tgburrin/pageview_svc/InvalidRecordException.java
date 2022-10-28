package net.tgburrin.pageview_svc;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ResponseStatus;

@SuppressWarnings("serial")
@ResponseStatus(value = HttpStatus.NOT_FOUND)
public class InvalidRecordException extends Exception {
	public InvalidRecordException (String message) {
		super(message);
	}

	public InvalidRecordException (String message, Throwable err) {
		super(message, err);
	}
}
